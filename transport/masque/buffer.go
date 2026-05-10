package masque

import "sync"

type NetBuffer struct {
	capacity uint32
	buf      sync.Pool
}

func (n *NetBuffer) Get() []byte {
	return *n.buf.Get().(*[]byte)
}

func (n *NetBuffer) Put(buf []byte) {
	if cap(buf) != int(n.capacity) {
		return
	}
	n.buf.Put(&buf)
}

func NewNetBuffer(capacity uint32) *NetBuffer {
	if capacity == 0 {
		panic("capacity must be greater than 0")
	}
	return &NetBuffer{
		capacity: capacity,
		buf: sync.Pool{
			New: func() interface{} {
				b := make([]byte, capacity)
				return &b
			},
		},
	}
}
