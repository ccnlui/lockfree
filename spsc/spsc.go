package spsc

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"
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
	position uint64
	data     interface{}
}

type nodes []node

type RingBuffer struct {
	_        [8]uint64
	write    uint64 // Shared, owned by producer.
	_        [8]uint64
	read     uint64 // Shared, owned by consumer.
	_        [8]uint64
	mask     uint64
	disposed uint64
	_        [8]uint64
	nodes    nodes
}

func (rb *RingBuffer) init(size uint64) {
	size = roundUp(size)
	rb.nodes = make(nodes, size)
	for i := uint64(0); i < size; i++ {
		rb.nodes[i] = node{position: i}
	}
	rb.mask = size - 1 // so we don't have to do this with every put/get operation
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

// Get will return the next item in the queue.  This call will block
// if the queue is empty.  This call will unblock when an item is added
// to the queue or Dispose is called on the queue.  An error will be returned
// if the queue is disposed.
func (rb *RingBuffer) Get() (interface{}, error) {
	return rb.Poll(0)
}

// Poll will return the next item in the queue.  This call will block
// if the queue is empty.  This call will unblock when an item is added
// to the queue, Dispose is called on the queue, or the timeout is reached. An
// error will be returned if the queue is disposed or a timeout occurs. A
// non-positive timeout will block indefinitely.
func (rb *RingBuffer) Poll(timeout time.Duration) (interface{}, error) {
	var start time.Time
	if timeout > 0 {
		start = time.Now()
	}

	rd := atomic.LoadUint64(&rb.read)
	for {
		if atomic.LoadUint64(&rb.disposed) > 0 {
			return nil, errors.New(`queue: closed`)
		}
		wr := atomic.LoadUint64(&rb.write)
		// Not emtpy.
		if rd != wr {
			break
		}
		if timeout > 0 && time.Since(start) >= timeout {
			return nil, errors.New(`queue: poll timed out`)
		}
		runtime.Gosched() // free up the cpu before the next iteration
	}
	n := &rb.nodes[rd&rb.mask]
	data := n.data
	n.data = nil
	atomic.StoreUint64(&rb.read, rd+1) // cache coherence traffic.
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
	wr := atomic.LoadUint64(&rb.write)
	for {
		if atomic.LoadUint64(&rb.disposed) > 0 {
			return false, errors.New(`queue: closed`)
		}
		rd := atomic.LoadUint64(&rb.read)
		// Not full.
		if wr < rd+rb.Cap() {
			break
		}
		if offer {
			return false, nil
		}
		runtime.Gosched() // free up the cpu before the next iteration
	}
	n := &rb.nodes[wr&rb.mask]
	n.data = item
	atomic.StoreUint64(&rb.write, wr+1) // cache coherence traffic.
	return true, nil
}
