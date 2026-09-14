package kvstore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/inqnhoang/byodb/engine/btree"
	"golang.org/x/sys/unix"
)

const DB_SIG = "BuildYourOwnDB06"

func assert(cond bool, caller string) {
	if !cond {
		panic(fmt.Sprintf("assertion failed: %s", caller))
	}
}

// -----=================-----
// ---=====  KvStore  =====---
// -----=================-----

type KV struct {
	Path string // file path
	fd   int    // file descriptor
	tree *btree.BTree
	free FreeList

	mmap struct {
		total  int // virtual addresses mapped (flushed & not-backed)
		chunks [][]byte
	}
	// whole multi-page operation
	page struct {
		flushed uint64            // # of pages flushed to disk
		temp    [][]byte          // pages queue
		updates map[uint64][]byte // pending updates, including appended pages
	}

	failed bool // Did last update fail?
}

func (db *KV) Get(key []byte) ([]byte, bool) {
	return db.tree.Get(key)
}

func (db *KV) Set(key []byte, val []byte) error {
	meta := saveMeta(db)
	db.tree.Insert(key, val)
	return updateOrRevert(db, meta)
}

func (db *KV) Del(key []byte) (bool, error) {
	deleted := db.tree.Delete(key)
	return deleted, updateFile(db)
}

// -----==============-----
// ---=====  Mmap  =====---
// -----==============-----

// extends capacity of mmap
func extendMmap(db *KV, size int) error {
	if size <= db.mmap.total {
		return nil
	}
	alloc := max(db.mmap.total, 64<<20) // double current address space or add 8MB
	for db.mmap.total+alloc < size {
		alloc *= 2
	}

	chunk, err := syscall.Mmap(
		db.fd, int64(db.mmap.total), alloc,
		syscall.PROT_READ, syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap: %w", err)
	}
	db.mmap.total += alloc
	db.mmap.chunks = append(db.mmap.chunks, chunk)
	return nil
}

// -----===============-----
// ---=====  Pages  =====---
// -----===============-----

// flush pages queue to disk
func writePages(db *KV) error {
	// extend mmap if needed
	// flushed + queue size
	size := (int(db.page.flushed) + len(db.page.temp)) * btree.BTREE_PAGE_SIZE
	if err := extendMmap(db, size); err != nil {
		return err
	}

	offset := int64(db.page.flushed * btree.BTREE_PAGE_SIZE)
	if _, err := unix.Pwritev(db.fd, db.page.temp, offset); err != nil {
		return err
	}

	db.page.flushed += uint64(len(db.page.temp))
	db.page.temp = db.page.temp[:0]
	return nil
}

func (db *KV) pageReadFile(ptr uint64) []byte {
	start := uint64(0)
	for _, chunk := range db.mmap.chunks {
		end := start + uint64(len(chunk))/btree.BTREE_PAGE_SIZE // page mapping
		if ptr < end {
			offset := btree.BTREE_PAGE_SIZE * (ptr - start)
			return chunk[offset : offset+btree.BTREE_PAGE_SIZE]
		}
		start = end
	}
	panic("bad ptr!")
}

// reads a page using a logical pointer
func (db *KV) pageRead(ptr uint64) []byte {
	if node, ok := db.page.updates[ptr]; ok {
		return node
	}
	return db.pageReadFile(ptr)
}

// appends a page to queue
func (db *KV) pageAppend(node []byte) uint64 {
	ptr := db.page.flushed + uint64(len(db.page.temp))
	db.page.temp = append(db.page.temp, node)
	return ptr
}

// allocates a page
func (db *KV) pageAlloc(node []byte) uint64 {
	// check for recycled
	if ptr := db.free.PopHead(); ptr != 0 {
		db.page.updates[ptr] = node
		return ptr
	}
	// otherwise append
	return db.pageAppend(node)
}

func (db *KV) pageWrite(ptr uint64) []byte {
	if node, ok := db.page.updates[ptr]; ok {
		return node
	}
	node := make([]byte, btree.BTREE_PAGE_SIZE)
	copy(node, db.pageReadFile(ptr))
	db.page.updates[ptr] = node
	return node
}

// -----==============-----
// ---=====  Meta  =====---
// -----==============-----

// | sig | root_ptr | page_used | head_page | head_seq | tail_page | tail_seq |
// | 16B |    8B    |     8B    |     8B    |    8B    |     8B    |    8B    |
func saveMeta(db *KV) []byte {
	var data [32]byte
	copy(data[:16], []byte(DB_SIG))
	binary.LittleEndian.PutUint64(data[16:], db.tree.GetRoot())
	binary.LittleEndian.PutUint64(data[24:], db.page.flushed)
	return data[:]
}

func loadMeta(db *KV, data []byte) error {
	if !bytes.Equal(data[:16], []byte(DB_SIG)) {
		return fmt.Errorf("wrong meta sig")
	}
	root := binary.LittleEndian.Uint64(data[16:])
	db.tree.SetRoot(root)
	db.page.flushed = binary.LittleEndian.Uint64(data[24:])
	db.free.headPage = binary.LittleEndian.Uint64(data[32:])
	db.free.headSeq = binary.LittleEndian.Uint64(data[40:])
	db.free.tailPage = binary.LittleEndian.Uint64(data[48:])
	db.free.tailSeq = binary.LittleEndian.Uint64(data[56:])
	return nil
}

func readRoot(db *KV, filesize uint64) error {
	if filesize == 0 {
		db.page.flushed = 1 // meta page
		db.free.headPage = 1
		db.free.tailPage = 1
		return nil
	}
	data := db.mmap.chunks[0]
	loadMeta(db, data)

	//  verify the page
	maxPages := filesize / btree.BTREE_PAGE_SIZE
	if db.tree.GetRoot() >= maxPages {
		return fmt.Errorf("readRoot: invalid root pointer %d, file has %d pages", db.tree.GetRoot(), maxPages)
	}
	return nil
}

func updateRoot(db *KV) error {
	if _, err := syscall.Pwrite(db.fd, saveMeta(db), 0); err != nil {
		return err
	}
	return nil
}

func updateFile(db *KV) error {
	if err := writePages(db); err != nil {
		return err
	}

	if err := syscall.Fsync(db.fd); err != nil {
		return err
	}

	if err := updateRoot(db); err != nil {
		return err
	}

	db.free.SetMaxSeq()
	return syscall.Fsync(db.fd)
}

// meta is from the in-memory state
func updateOrRevert(db *KV, meta []byte) error {
	// before updating, if the db has failed a previous update,
	// revert to the old data, rather than retrying after the intial update fail
	if db.failed {
		if _, err := syscall.Pwrite(db.fd, meta, 0); err != nil {
			return fmt.Errorf("updateOrRevert: repair Pwrite")
		}

		if err := syscall.Fsync(db.fd); err != nil {
			return fmt.Errorf("updateOrRevert: repair Fsync")
		}
		db.failed = false
	}

	// 2-phase update
	err := updateFile(db)
	// revert to in-memory
	if err != nil {
		loadMeta(db, meta)
		db.page.temp = db.page.temp[:0]
		db.failed = true
	}
	return err
}

func (db *KV) Open() error {
	db.tree.SetGet(db.pageRead)
	db.tree.SetNew(db.pageAlloc)
	db.tree.SetDel(db.free.PushTail)

	db.free.get = db.pageRead
	db.free.new = db.pageAppend
	db.free.set = db.pageWrite
	return nil
}
