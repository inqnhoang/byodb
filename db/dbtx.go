package db

import "github.com/inqnhoang/byodb/engine/kvstore"

type DBTX struct {
	kv kvstore.KVTX
	db *DB
}

func (tx *DBTX) Scan(table string, req *Scanner) error
func (tx *DBTX) Set(table string, rec Record, mode int)
func (tx *DBTX) Delete(table string, rec Record) (bool, error)
