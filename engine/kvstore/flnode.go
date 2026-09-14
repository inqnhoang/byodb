package kvstore

// | next | pointers |
// |  8B  |   8B...  |
type LNode []byte

func (node LNode) getNext() uint64
func (node LNode) setNext(next uint64)
func (node LNode) getPtr(idx int) uint64
func (node LNode) setPtr(idx int, ptr uint64)
