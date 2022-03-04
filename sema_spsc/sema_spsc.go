package sema_spsc

import (
	"errors"
	"sync/atomic"
)

// roundUp takes a uint64 greater than 0 and rounds it up to the next
// power of 2.
func roundUp(v uint64) uint64 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++
	return v
}

type node struct {
	_      [8]uint64
	semaWr int32 // Shared. Number of available writes.
	semaRd int32 // Shared. Number of available reads.
	_      [8]uint64
	data   interface{}
	ch     chan struct{}
}

type nodes []node

// RingBuffer is a SPSC lockfree queue. This implementation is based on Dmitry's
// bounded mpmc queue from https://www.1024cores.net/home/lock-free-algorithms/queues/bounded-mpmc-queue.
type RingBuffer struct {
	_        [8]uint64
	write    uint64 // Not shared, owned by producer.
	_        [8]uint64
	read     uint64 // Not shared, owned by consumer.
	_        [8]uint64
	mask     uint64
	disposed uint64
	_        [8]uint64
	nodes    nodes
}

func (rb *RingBuffer) init(size uint64) {
	size = roundUp(size)
	rb.mask = size - 1 // so we don't have to do this with every put/get operation
	rb.nodes = make(nodes, size)
	for i := range rb.nodes {
		atomic.StoreInt32(&rb.nodes[i].semaWr, 1)
		rb.nodes[i].ch = make(chan struct{}, 1)
	}
}

// NewRingBuffer will allocate, initialize, and return a ring buffer
// with the specified size.
func NewRingBuffer(size uint64) *RingBuffer {
	rb := &RingBuffer{}
	rb.init(size)
	return rb
}

// Dispose will dispose of this queue and free any blocked threads
// in the Put and/or Get methods.  Calling those methods on a disposed
// queue will return an error.
func (rb *RingBuffer) Dispose() {
	atomic.CompareAndSwapUint64(&rb.disposed, 0, 1)
}

// IsDisposed will return a bool indicating if this queue has been
// disposed.
func (rb *RingBuffer) IsDisposed() bool {
	return atomic.LoadUint64(&rb.disposed) == 1
}

// Cap returns the capacity of this ring buffer.
func (rb *RingBuffer) Cap() uint64 {
	return uint64(len(rb.nodes))
}

// Len returns the length of this ring buffer.
func (rb *RingBuffer) Len() uint64 {
	return rb.write - rb.read
}

func (rb *RingBuffer) Get() (interface{}, error) {
	n := &rb.nodes[rb.read&rb.mask]
	if atomic.LoadUint64(&rb.disposed) == 1 {
		return nil, errors.New(`queue: closed`)
	}

	// Semaphore wait.
	rd := atomic.AddInt32(&n.semaRd, -1) // cache coherence traffic
	if rd < 0 {
		<-n.ch // queue is empty, sleep now
	}

	rb.read++
	data := n.data

	// Semaphore signal.
	wr := atomic.AddInt32(&n.semaWr, 1) // cache coherence traffic
	if wr < 1 {
		n.ch <- struct{}{} // queue was full, wake up other goroutine
	}

	return data, nil
}

// Put adds the provided item to the queue.  If the queue is full, this
// call will block until an item is added to the queue or Dispose is called
// on the queue.  An error will be returned if the queue is disposed.
func (rb *RingBuffer) Put(item interface{}) error {
	_, err := rb.put(item, false)
	return err
}

// Offer adds the provided item to the queue if there is space.  If the queue
// is full, this call will return false.  An error will be returned if the
// queue is disposed.
func (rb *RingBuffer) Offer(item interface{}) (bool, error) {
	return rb.put(item, true)
}

func (rb *RingBuffer) put(item interface{}, offer bool) (bool, error) {
	n := &rb.nodes[rb.write&rb.mask]
	if atomic.LoadUint64(&rb.disposed) == 1 {
		return false, errors.New(`queue: closed`)
	}

	// Semaphore wait.
	wr := atomic.AddInt32(&n.semaWr, -1) // cache coherence traffic
	if wr < 0 {
		<-n.ch // queue is full, sleep now
	}

	rb.write++
	n.data = item

	// Semaphore signal.
	rd := atomic.AddInt32(&n.semaRd, 1) // cache coherence traffic
	if rd < 1 {
		n.ch <- struct{}{} // queue was empty, wake up other goroutine
	}

	return true, nil
}
