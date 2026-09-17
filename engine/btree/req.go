package btree

const (
	MODE_UPSERT      = 0
	MODE_UPDATE_ONLY = 1
	MODE_INSERT_ONLY = 2
)

type UpdateReq struct {
	tree *BTree

	Added   bool
	Updated bool
	Old     []byte

	Key  []byte
	Val  []byte
	Mode int
}
