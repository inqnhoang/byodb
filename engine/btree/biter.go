package btree

type BIter struct {
	tree *BTree
	path []BNode
	pos  []uint16
}

func (iter *BIter) Deref() ([]byte, []byte) {
	last := len(iter.pos) - 1
	pos := iter.pos[last]
	node := BNode(iter.path[last])
	return node.getKey(pos), node.getVal(pos)
}

func (iter *BIter) Valid() bool {
	last := len(iter.pos) - 1
	pos := iter.pos[last]
	return pos > 0 && pos < iter.path[last].nkeys()
}

func iterPrev(iter *BIter, level int) {
	if iter.pos[level] > 0 {
		iter.pos[level]--
	} else if level > 0 {
		iterPrev(iter, level-1)
	} else {
		iter.pos[len(iter.pos)-1]-- // underflow
		return
	}

	if level+1 < len(iter.pos) {
		node := iter.path[level]
		kid := BNode(iter.tree.get(node.getPtr(iter.pos[level])))
		iter.path[level+1] = kid
		iter.pos[level+1] = kid.nkeys() - 1
	}
}

func (iter *BIter) Prev() {
	iterPrev(iter, len(iter.path)-1)
}

func (iter *BIter) Next() {
	iterNext(iter, len(iter.path)-1)
}

func iterNext(iter *BIter, level int) {
	if iter.pos[level]+1 < iter.path[level].nkeys() {
		iter.pos[level]++
	} else if level > 0 {
		iterNext(iter, level-1)
	} else {
		iter.pos[len(iter.pos)-1]++
		return
	}
	if level+1 < len(iter.pos) { // all internal nodes
		node := iter.path[level] // advance an internal node
		kid := BNode(iter.tree.get(node.getPtr(iter.pos[level])))
		iter.path[level+1] = kid
		iter.pos[level+1] = 0
	}
}
