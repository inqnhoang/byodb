package btree

import "fmt"

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

const (
	CMP_GE = +3
	CMP_GT = +2
	CMP_LT = -2
	CMP_LE = -3
)

type BTree struct {
	root uint64 // pointer to a non-zero page number

	// call-backs for managing on-disk pages
	get func(uint64) []byte
	new func([]byte) uint64 // copy-on-write
	del func(uint64)
}

func (tree *BTree) GetRoot() uint64 {
	return tree.root
}

func (tree *BTree) SetRoot(root uint64) {
	tree.root = root
}

func (tree *BTree) SetGet(get func(uint64) []byte) {
	tree.get = get
}

func (tree *BTree) SetNew(new func([]byte) uint64) {
	tree.new = new
}

func (tree *BTree) SetDel(del func(uint64)) {
	tree.del = del
}

func (tree *BTree) Get(key []byte) ([]byte, bool) {
	if tree.root == 0 {
		return nil, false
	}
	return treeGet(tree, tree.get(tree.root), key)
}

// insertion into tree by building new nodes for the path it goes on
func (tree *BTree) Insert(key []byte, val []byte, mode int) {
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
	node := treeInsert(tree, tree.get(tree.root), key, val, mode)

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
	tree.del(tree.root)

	if node.btype() == BNODE_LEAF && node.nkeys() == 1 { // reset btree
		tree.root = 0
	} else if node.btype() == BNODE_NODE && node.nkeys() == 1 { // if root only has one child, just make that child the root
		tree.root = node.getPtr(0)
	} else {
		tree.root = tree.new(node)
	}
	return true
}

func treeSeek(tree *BTree, key []byte, lookup func(BNode, []byte) uint16) *BIter {
	iter := &BIter{tree: tree}
	for ptr := tree.root; ptr != 0; {
		node := BNode(tree.get(ptr))
		idx := lookup(node, key)

		if node.btype() == BNODE_LEAF {
			break
		}

		if idx >= node.nkeys() {
			break
		}
		iter.path = append(iter.path, node)
		iter.pos = append(iter.pos, idx)
		ptr = node.getPtr(idx)
	}
	return iter
}

func (tree *BTree) SeekGE(key []byte) *BIter { return treeSeek(tree, key, nodeLookupGE) }
func (tree *BTree) SeekGT(key []byte) *BIter { return treeSeek(tree, key, nodeLookupGT) }
func (tree *BTree) SeekLE(key []byte) *BIter { return treeSeek(tree, key, nodeLookupLE) }
func (tree *BTree) SeekLT(key []byte) *BIter { return treeSeek(tree, key, nodeLookupLT) }

func (tree *BTree) Seek(key []byte, cmp int) *BIter {
	switch cmp {
	case CMP_GE:
		return tree.SeekGE(key)
	case CMP_GT:
		return tree.SeekGT(key)
	case CMP_LE:
		return tree.SeekLE(key)
	case CMP_LT:
		return tree.SeekLT(key)
	default:
		panic(fmt.Errorf("Seek: bad cmp %d", cmp))
	}
}
