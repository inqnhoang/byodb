package db

import (
	"slices"

	"github.com/inqnhoang/byodb/engine/btree"
)

type Scanner struct {
	// range from k1, k2
	Cmp1 int
	Cmp2 int
	Key1 Record
	Key2 Record

	db     *DB
	tdef   *TableDef
	index  int
	iter   *btree.BIter
	keyEnd []byte
}

// TODO
func (sc *Scanner) Valid()
func (sc *Scanner) Next()
func (sc *Scanner) Deref()

func dbScan(db *DB, tdef *TableDef, req *Scanner) {
	isCovered := func(index []string) bool {
		key := req.Key1.Cols
		return len(index) >= len(key) && slices.Equal(index[:len(key)], key)
	}
	req.index = slices.IndexFunc(tdef.Indexes, isCovered)
}
