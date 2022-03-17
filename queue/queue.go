package queue

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
)

// Node struct
type Node struct {
	Item interface{}
	Next *Node
}

// Queue struct
type Queue struct {
	sync.Mutex
	Count    int32
	Capacity int32
	Head     *Node
	Last     *Node
}

func NewQueue() *Queue {
	node := &Node{}
	return &Queue{
		Count:    int32(0),
		Capacity: math.MaxInt32,
		Head:     node,
		Last:     node,
	}
}

// Put item to queue
func (q *Queue) Put(item interface{}) error {
	if item == nil {
		return errors.New("item can't be nil")
	}

	getPos := atomic.LoadInt32(&q.Count)
	if getPos >= q.Capacity {
		return errors.New("queue size exceeding maximum capacity")
	}

	q.Lock()
	defer q.Unlock()

	node := &Node{Item: item}
	q.Last.Next = node
	q.Last = node

	atomic.AddInt32(&q.Count, 1)
	return nil
}

// Poll 保证单协程执行，不加锁
func (q *Queue) Poll() (has bool, item interface{}) {
	node := q.Head.Next
	if node == nil {
		return false, nil
	}

	res := node.Item
	node.Item = nil // help GC

	q.Head = node
	atomic.AddInt32(&q.Count, -1)
	return true, res
}

func (q *Queue) Clear() {
	q = NewQueue()
}

// HasNext check
func (q *Queue) HasNext() bool {
	return atomic.LoadInt32(&q.Count) > 0
}
