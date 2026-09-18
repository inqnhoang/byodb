package kvstore

import (
	"github.com/inqnhoang/byodb/engine/btree"
)

const FREE_LIST_HEADER = 8
const FREE_LIST_CAP = (btree.BTREE_PAGE_SIZE - FREE_LIST_HEADER) / 8

type FreeList struct {
	get func(uint64) []byte
	new func([]byte) uint64
	set func(uint64) []byte

	headPage uint64 // idx
	headSeq  uint64
	tailPage uint64
	tailSeq  uint64

	maxSeq uint64
	maxVer uint64
	curVer uint64
}

func seq2idx(seq uint64) int {
	return int(seq % FREE_LIST_CAP)
}

func (fl *FreeList) SetMaxSeq() {
	fl.maxSeq = fl.tailSeq
}

// returns a freed page within the head node, and returns the head node
// if it's empty
func flPop(fl *FreeList) (ptr uint64, head uint64) {
	// empty
	if fl.headSeq == fl.tailSeq {
		return 0, 0
	}
	node := LNode(fl.get(fl.headPage))
	ptr = node.getPtr(seq2idx(fl.headSeq))
	fl.headSeq++

	if seq2idx(fl.headSeq) == 0 { // page is empty
		head, fl.headPage = fl.headPage, node.getNext()
		assert(fl.headPage != 0, "flPop") // the next page is retrieved
	}
	return
}

func (fl *FreeList) PopHead() uint64 {
	ptr, head := flPop(fl)
	if head != 0 {
		fl.PushTail(head)
	}
	return ptr
}

func (fl *FreeList) PushTail(ptr uint64) {
	// safe, new tailPage is assigned when previous gets full
	LNode(fl.set(fl.tailPage)).setPtr(seq2idx(fl.tailSeq), ptr)
	fl.tailSeq++

	// if page is full now, create the next page
	if seq2idx(fl.tailSeq) == 0 {
		// take from the head - an used page
		next, head := flPop(fl)

		// dead code?
		if next == 0 {
			next = fl.new(make([]byte, btree.BTREE_PAGE_SIZE))
		}

		// set the tailpage as the recycled unused page
		LNode(fl.set(fl.tailPage)).setNext(next)
		fl.tailPage = next

		// if flpop returned the node as a whole, set the first unused page
		// of next to be head
		if head != 0 {
			LNode(fl.set(fl.tailPage)).setPtr(0, head)
			fl.tailSeq++
		}
	}
}
