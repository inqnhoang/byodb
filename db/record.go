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

func encodeKey(out []byte, prefix uint32, vals []Value) []byte {
	var buf [4]byte
	binary.LittleEndian.AppendUint32(buf[:], prefix)
	out = append(out, buf[:]...)

	for _, val := range vals {
		switch val.Type {
		case TYPE_INT64:
			var b [8]byte
			binary.LittleEndian.AppendUint64(b[:], uint64(val.I64))
			out = append(out, b[:]...)

		// | size | str |
		// |  4B  | ... |
		case TYPE_BYTES:
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(len(val.Str)))
			out = append(out, b[:]...)
			out = append(out, val.Str...)
		default:
			panic(fmt.Errorf("encodeKey: unknown type %d", val.Type))
		}
	}
	return out
}

func decodeValues(in []byte, out []Value) {
	offset := 0
	for i, val := range out {
		switch val.Type {
		case TYPE_INT64:
			out[i].I64 = int64(binary.LittleEndian.Uint64(in[offset:]))
			offset += 8

		// | size | str |
		// |  4B  | ... |
		case TYPE_BYTES:
			str_len := binary.LittleEndian.Uint32(in[offset:])
			offset += 4
			out[i].Str = append([]byte{}, in[offset:offset+int(str_len)]...)
			offset += int(str_len)
		default:
			panic(fmt.Errorf("encodeKey: unknown type %d", val.Type))
		}
	}
}
