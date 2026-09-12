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

// -----=================-----
// ---=====  KvStore  =====---
// -----=================-----

type KV struct {
	Path string // file path
	fd   int    // file descriptor
	tree *btree.BTree

	mmap struct {
		total  int // virtual addresses mapped (flushed & not-backed)
		chunks [][]byte
	}
	// whole multi-page operation
	page struct {
		flushed uint64 // # of pages flushed to disk
		temp    [][]byte
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

// virtual address
func (db *KV) pageRead(ptr uint64) []byte {
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

func (db *KV) pageAppend(node []byte) uint64 {
	ptr := db.page.flushed + uint64(len(db.page.temp))
	db.page.temp = append(db.page.temp, node)
	return ptr
}

// -----==============-----
// ---=====  Meta  =====---
// -----==============-----

// | sig | root_ptr | page_used |
// | 16B |    8B    |     8B    |
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
	return nil
}

func readRoot(db *KV, filesize uint64) error {
	if filesize == 0 {
		db.page.flushed = 1 // meta page
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
	db.tree.SetNew(db.pageAppend)
	db.tree.SetDel(func(uint64) {})
	return nil
}
