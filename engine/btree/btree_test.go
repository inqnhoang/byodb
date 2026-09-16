package btree

import (
	"fmt"
	"math/rand"
	"testing"
	"unsafe"
)

type C struct {
	tree  BTree
	ref   map[string]string
	pages map[uint64]BNode
}

func newC() *C {
	pages := map[uint64]BNode{}
	return &C{
		tree: BTree{
			root: 0,
			get: func(ptr uint64) []byte {
				node, ok := pages[ptr]
				assert(ok, "btree.get")
				return node
			},
			new: func(node []byte) uint64 {
				assert(BNode(node).nbytes() <= BTREE_PAGE_SIZE, "btree.new")
				ptr := uint64(uintptr(unsafe.Pointer(&node[0])))
				_, exists := pages[ptr]
				assert(!exists, "btree.new")
				pages[ptr] = node
				return ptr
			},
			del: func(ptr uint64) {
				_, exists := pages[ptr]
				assert(exists, "btree.del")
				delete(pages, ptr)
			},
		},
		ref:   map[string]string{},
		pages: pages,
	}
}

func (c *C) add(key string, val string) {
	c.tree.Insert([]byte(key), []byte(val), MODE_UPSERT)
	c.ref[key] = val
}

func (c *C) del(key string) bool {
	deleted := c.tree.Delete([]byte(key))
	if deleted {
		delete(c.ref, key)
	}
	return deleted
}

func (c *C) dump() map[string]string {
	got := map[string]string{}
	var walk func(ptr uint64)
	walk = func(ptr uint64) {
		node := BNode(c.tree.get(ptr))
		if node.btype() == BNODE_LEAF {
			for i := uint16(0); i < node.nkeys(); i++ {
				k := node.getKey(i)
				if len(k) == 0 {
					continue // dummy node
				}
				v := node.getVal(i)
				got[string(k)] = string(v)
			}
			return
		}
		for i := uint16(0); i < node.nkeys(); i++ {
			walk(node.getPtr(i))
		}
	}
	if c.tree.root != 0 {
		walk(c.tree.root)
	}
	return got
}

func (c *C) verify(t *testing.T) {
	t.Helper()
	got := c.dump()
	if len(got) != len(c.ref) {
		t.Fatalf("size mismatch: tree has %d entries, ref has %d", len(got), len(c.ref))
	}
	for k, v := range c.ref {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("missing key %q in tree", k)
		}
		if gv != v {
			t.Fatalf("value mismatch for key %q: tree=%q ref=%q", k, gv, v)
		}
	}
}

func TestInsertAndGet(t *testing.T) {
	c := newC()
	c.add("k1", "hello")
	c.add("k2", "world")
	c.verify(t)
}

func TestUpdateExistingKey(t *testing.T) {
	c := newC()
	c.add("key", "v1")
	c.add("key", "v2")
	c.verify(t)
	if len(c.ref) != 1 {
		t.Fatalf("expected 1 entry after update, ref has %d", len(c.ref))
	}
}

func TestDeleteExisting(t *testing.T) {
	c := newC()
	c.add("a", "1")
	c.add("b", "2")
	c.add("c", "3")

	if !c.del("b") {
		t.Fatalf("expected delete of existing key to return true")
	}
	c.verify(t)
}

func TestDeleteMissingKey(t *testing.T) {
	c := newC()
	c.add("a", "1")

	if c.del("does-not-exist") {
		t.Fatalf("expected delete of missing key to return false")
	}
	// tree must be completely unchanged
	c.verify(t)
}

func TestDeleteFromEmptyTree(t *testing.T) {
	c := newC()
	if c.del("anything") {
		t.Fatalf("expected delete on empty tree to return false")
	}
}

func TestSplitOnManyInserts(t *testing.T) {
	c := newC()
	n := 500
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%05d", i)
		val := fmt.Sprintf("val-%05d", i)
		c.add(key, val)
	}
	c.verify(t)
}

func TestSplitThenDeleteAll(t *testing.T) {
	c := newC()
	n := 300
	keys := make([]string, 0, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%05d", i)
		val := fmt.Sprintf("val-%05d", i)
		c.add(key, val)
		keys = append(keys, key)
	}
	c.verify(t)

	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	for _, k := range keys {
		if !c.del(k) {
			t.Fatalf("expected delete of %q to succeed", k)
		}
		c.verify(t)
	}

	if c.tree.root != 0 {
		t.Fatalf("expected empty tree (root == 0) after deleting everything, got root=%d", c.tree.root)
	}
}

func TestLargeValueNearPageLimit(t *testing.T) {
	c := newC()
	bigVal := make([]byte, BTREE_PAGE_SIZE/4-10)
	for i := range bigVal {
		bigVal[i] = byte('x')
	}
	c.add("bigkey", string(bigVal))
	c.verify(t)
}

func TestRandomInsertDeleteMix(t *testing.T) {
	c := newC()
	rng := rand.New(rand.NewSource(42))
	present := map[string]bool{}

	for i := 0; i < 2000; i++ {
		key := fmt.Sprintf("k-%03d", rng.Intn(300))
		if rng.Intn(2) == 0 || !present[key] {
			val := fmt.Sprintf("v-%d", rng.Int())
			c.add(key, val)
			present[key] = true
		} else {
			c.del(key)
			present[key] = false
		}
	}
	c.verify(t)
}

func TestSequentialAscendingInserts(t *testing.T) {
	c := newC()
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("%06d", i)
		c.add(key, "v")
	}
	c.verify(t)
}

func TestSequentialDescendingInserts(t *testing.T) {
	c := newC()
	for i := 999; i >= 0; i-- {
		key := fmt.Sprintf("%06d", i)
		c.add(key, "v")
	}
	c.verify(t)
}
