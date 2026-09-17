package db

import "encoding/json"

type TableDef struct {
	Name     string
	Types    []uint32
	Cols     []string
	Indexes  [][]string
	Prefixes []uint32
}

var TDEF_META = &TableDef{
	Prefixes: []uint32{1},
	Name:     "@meta",
	Types:    []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:     []string{"key", "val"},
}

var TDEF_TABLE = &TableDef{
	Prefixes: []uint32{2},
	Name:     "@table",
	Types:    []uint32{TYPE_BYTES, TYPE_BYTES},
	Cols:     []string{"name", "def"},
}

func getTableDef(db *DB, name string) *TableDef {
	rec := (&Record{}).AddStr("name", []byte(name))
	ok, err := dbGet(db, TDEF_TABLE, rec)
	assert(err == nil, "getTabelDef: dbGet")
	if !ok {
		return nil
	}
	tdef := &TableDef{}
	err = json.Unmarshal(rec.Get("def").Str, tdef)
	assert(err == nil, "getTableDef: json.unmarshal")
	return tdef
}
