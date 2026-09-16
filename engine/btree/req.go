package btree

const (
	MODE_UPSERT      = 0
	MODE_UPDATE_ONLY = 1
	MODE_INSERT_ONLY = 2
)

type UpdateReq struct {
	Mode int

	tree  *BTree
	Added bool
	Key   []byte
	Val   []byte
}
