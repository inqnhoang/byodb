package kvstore

import (
	"github.com/inqnhoang/byodb/engine/btree"
)

type KV struct {
	Path string
	fd   int

	tree btree.BTree
}

func (db *KV) Open() error

func (db *KV) Get(key []byte) ([]byte, bool) {
	return db.tree.Get(key)
}

func (db *KV) Set(key []byte, val []byte) error {
	db.tree.Insert(key, val)
	return nil
}

func (db *KV) Del(key []byte) (bool, error) {
	db.tree.Delete(key)
	return false, nil
}
