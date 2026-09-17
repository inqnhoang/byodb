package db

import (
	"encoding/binary"
	"fmt"
)

const (
	TYPE_BYTES = 1
	TYPE_INT64 = 2
)

type Value struct {
	Type uint32 // tagged union
	I64  int64
	Str  []byte
}

type Record struct {
	Cols []string
	Vals []Value
}

func (rec *Record) AddStr(col string, val []byte) *Record {
	rec.Cols = append(rec.Cols, col)
	rec.Vals = append(rec.Vals, Value{Type: TYPE_BYTES, Str: val})
	return rec
}

func (rec *Record) AddInt64(col string, val int64) *Record {
	rec.Cols = append(rec.Cols, col)
	rec.Vals = append(rec.Vals, Value{Type: TYPE_BYTES, I64: val})
	return rec
}

func (rec *Record) Get(col string) *Value {
	for i, rcol := range rec.Cols {
		if rcol == col {
			return &rec.Vals[i]
		}
	}
	return nil
}

// reorder a record and check for missing columns
func checkRecord(tdef *TableDef, rec Record, n int) ([]Value, error) {
	vals := make([]Value, len(tdef.Cols))
	for i, col := range tdef.Cols {
		v := rec.Get(col)
		if v == nil {
			if i < n {
				return nil, fmt.Errorf("checkRecord: missing primary key %s", col)
			}
			continue
		}

		if v.Type != tdef.Types[i] {
			return nil, fmt.Errorf("checkRecord: bad type for column %s", col)
		}
		vals[i] = *v
	}
	return vals, nil
}

// -----======================-----
// ---===== Encode & Decode =====--
// -----=======================----

func escapeString(in []byte) []byte {
	out := make([]byte, 0, len(in)+1)
	for _, b := range in {
		if b <= 1 {
			out = append(out, 0x01, b+1)
		} else {
			out = append(out, b)
		}
	}
	return out
}

func unescapeString(in []byte) []byte

// TODO
func encodeValues(out []byte, vals []Value) []byte {
	for _, v := range vals {
		out = append(out, byte(v.Type))
		switch v.Type {
		case TYPE_INT64:
			var buf [8]byte
			u := uint64(v.I64) + (1 << 63)
			binary.BigEndian.PutUint64(buf[:], u)
			out = append(out, buf[:]...)
		case TYPE_BYTES:
			out = append(out, escapeString(v.Str)...)
			out = append(out, 0)
		default:
			panic("what?")
		}
	}
	return out
}
func decodeValues(in []byte, out []Value)

func encodeKey(out []byte, prefix uint32, vals []Value) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], prefix)
	out = append(out, buf[:]...)
	out = encodeValues(out, vals)
	return out
}

func decodeKey(in []byte, out []Value)
