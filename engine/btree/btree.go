package btree

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

type BTree struct {
	root uint64 // pointer to a non-zero page number

	// call-backs for managing on-disk pages
	get func(uint64) []byte
	new func([]byte) uint64 // copy-on-write
	del func(uint64)
}
