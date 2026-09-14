package db

import (
	"fmt"

	kv "github.com/inqnhoang/byodb/engine/kvstore"
)

func assert(cond bool, caller string) {
	if !cond {
		panic(fmt.Sprintf("assertion failed: %s", caller))
	}
}

type DB struct {
	Path string
	kv   kv.KV
}

// primary keys are filled in rec
func dbGet(db *DB, tdef *TableDef, rec *Record) (bool, error) {
	values, err := checkRecord(tdef, *rec, tdef.PKeys)
	if err != nil {
		return false, err
	}
	key := encodeKey(nil, tdef.Prefix, values[:tdef.PKeys]) // encode prefix & primary keys

	val, ok := db.kv.Get(key) // get row
	if !ok {
		return false, nil
	}
	for i := tdef.PKeys; i < len(tdef.Cols); i++ {
		values[i].Type = tdef.Types[i]
	}
	decodeValues(val, values[tdef.PKeys:])
	rec.Cols = tdef.Cols
	rec.Vals = values
	return true, nil
}

// primary keys are filled in rec
func (db *DB) Get(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("Get: table not found %s", table)
	}
	return dbGet(db, tdef, rec)
}

func (db *DB) Insert(table string, rec Record) (bool, error)
func (db *DB) Update(table string, rec Record) (bool, error)
func (db *DB) Delete(table string, rec Record) (bool, error)
func (db *DB) Upsert(table string, rec Record) (bool, error)
