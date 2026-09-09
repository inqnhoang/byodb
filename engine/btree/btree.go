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

// insertion into tree by building new nodes for the path it goes on
func (tree *BTree) Insert(key []byte, val []byte) {
	// base case: root
	if tree.root == 0 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_LEAF, 2)

		nodeAppendKV(root, 0, 0, nil, nil)
		nodeAppendKV(root, 1, 0, key, val)
		tree.root = tree.new(root)
		return
	}

	// recursively handled, returns root' and only root', children are handled in the recursive calls
	node := treeInsert(tree, tree.get(tree.root), key, val)

	// check if root needs to get split
	nsplit, split := nodeSplit3(node)
	tree.del(tree.root)

	if nsplit > 1 {
		root := BNode(make([]byte, BTREE_PAGE_SIZE))
		root.setHeader(BNODE_NODE, nsplit)

		for i, knode := range split[:nsplit] {
			ptr, key := tree.new(knode), knode.getKey(0)
			nodeAppendKV(root, uint16(i), ptr, key, nil)
		}
		tree.root = tree.new(root)
	} else {
		tree.root = tree.new(split[0])
	}
}

func (tree *BTree) Delete(key []byte) bool {
	if tree.root == 0 {
		return false
	}

	// root'
	node := treeDelete(tree, tree.get(tree.root), key)
	if len(node) == 0 {
		return false
	}

	if node.btype() == BNODE_NODE && node.nkeys() == 1 {
		tree.root = node.getPtr(0) // if root only has one child, just make that child the root
	} else {
		tree.root = tree.new(node)
	}
	return true
}
