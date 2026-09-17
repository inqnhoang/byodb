package db

import (
	"fmt"

	"github.com/inqnhoang/byodb/engine/btree"
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
	nPkeys := len(tdef.Indexes[0])
	// reorder values
	values, err := checkRecord(tdef, *rec, nPkeys)
	if err != nil {
		return false, err
	}
	key := encodeKey(nil, tdef.Prefixes[0], values[:nPkeys]) // encode prefix & primary keys

	val, ok := db.kv.Get(key) // get row
	if !ok {
		return false, nil
	}
	for i := nPkeys; i < len(tdef.Cols); i++ {
		values[i].Type = tdef.Types[i]
	}
	decodeValues(val, values[nPkeys:])
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

func dbUpdate(db *DB, tdef *TableDef, rec *Record, mode int) (bool, error) {
	nPkeys := len(tdef.Indexes[0])
	values, err := checkRecord(tdef, *rec, nPkeys)
	if err != nil {
		return false, err
	}
	key := encodeKey(nil, tdef.Prefixes[0], values[:nPkeys])
	val := encodeValues(nil, values)

	req := btree.UpdateReq{Key: key, Val: val, Mode: mode}
	if _, err := db.kv.Update(&req); err != nil {
		return false, err
	}

	if req.Updated && !req.Added {
		// delete old indexed keys
	}

	if req.Updated {
		// add new index key
	}
	return req.Updated, nil
}

func (db *DB) Insert(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("Get: table not found %s", table)
	}
	return dbUpdate(db, tdef, rec, btree.MODE_INSERT_ONLY)
}

func (db *DB) Update(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("Get: table not found %s", table)
	}
	return dbUpdate(db, tdef, rec, btree.MODE_UPDATE_ONLY)
}

func dbDelete(db *DB, tdef *TableDef, rec *Record) (bool, error) {
	nPkeys := len(tdef.Indexes[0])
	values, err := checkRecord(tdef, *rec, nPkeys)
	if err != nil {
		return false, err
	}
	key := encodeKey(nil, tdef.Prefixes[0], values[:nPkeys])

	return db.kv.Del(key)
}

func (db *DB) Delete(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("Get: table not found %s", table)
	}
	return dbDelete(db, tdef, rec)
}

func (db *DB) Upsert(table string, rec *Record) (bool, error) {
	tdef := getTableDef(db, table)
	if tdef == nil {
		return false, fmt.Errorf("Get: table not found %s", table)
	}
	return dbUpdate(db, tdef, rec, btree.MODE_UPSERT)
}

func (db *DB) Scan(table string, req *Scanner) error
