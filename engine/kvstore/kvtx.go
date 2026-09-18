package kvstore

import (
	"github.com/inqnhoang/byodb/engine/btree"
)

const (
	FLAG_UPDATED = byte(1)
	FLAG_DELETED = byte(2)
)

// start <= key <= stop
type KeyRange struct {
	start []byte
	stop  []byte
}

type CommitedTX struct {
	version uint64
	writes  []KeyRange // sorted
}

type KVTX struct {
	db   *KV
	meta []byte

	pending  btree.BTree
	snapshot btree.BTree
	version  uint64

	reads []KeyRange
}

// Point-Query
func (tx *KVTX) Get(key []byte) ([]byte, bool) {
	tx.reads = append(tx.reads, KeyRange{key, key})
	val, ok := tx.pending.Get(key)
	switch {
	case ok && val[0] == FLAG_UPDATED:
		return val[1:], true
	case ok && val[0] == FLAG_DELETED:
		return nil, false
	case !ok:
		return tx.snapshot.Get(key)
	default:
		panic("unreachable")
	}
}
