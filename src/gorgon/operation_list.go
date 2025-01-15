package gorgon

import (
	"sync"
	"time"

	"github.com/anishathalye/porcupine"
)

type OperationList struct {
	head      *operationListNode
	startTime time.Time
	time      int64
	mutex     sync.Mutex
}

type operationListNode struct {
	next  *operationListNode
	value porcupine.Operation
}

func NewOperationList() *OperationList {
	return &OperationList{startTime: time.Now()}
}

func (list *OperationList) GetTime() int64 {
	now := int64(time.Since(list.startTime) / time.Microsecond)
	list.mutex.Lock()
	if now <= list.time {
		now = list.time + 1
	}
	list.time = now
	list.mutex.Unlock()
	return now
}

func (list *OperationList) Append(op porcupine.Operation) {
	node := &operationListNode{value: op}
	list.mutex.Lock()
	node.next = list.head
	list.head = node
	list.mutex.Unlock()
}

func (list *OperationList) Extract() []porcupine.Operation {
	list.mutex.Lock()
	head := list.head
	list.head = nil
	list.mutex.Unlock()

	var ret []porcupine.Operation
	for ; head != nil; head = head.next {
		ret = append(ret, head.value)
	}

	// Reverse
	for i, j := 0, len(ret)-1; i < j; {
		t := ret[i]
		ret[i] = ret[j]
		ret[j] = t
		i++
		j--
	}
	return ret
}
