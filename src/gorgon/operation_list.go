package gorgon

import (
	"sync"
	"time"
)

type OperationList struct {
	head      *operationListNode
	startTime time.Time
	time      int64
	mutex     sync.Mutex
}

type operationListNode struct {
	next  *operationListNode
	value Operation
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

func (list *OperationList) Append(op Operation) {
	node := &operationListNode{value: op}
	list.mutex.Lock()
	node.next = list.head
	list.head = node
	list.mutex.Unlock()
}

func (list *OperationList) Extract() []Operation {
	list.mutex.Lock()
	head := list.head
	list.head = nil
	list.mutex.Unlock()

	var ret []Operation
	var maxTime int64
	for head != nil {
		if maxTime < head.value.Return {
			maxTime = head.value.Return
		}
		if maxTime < head.value.Call {
			maxTime = head.value.Call
		}
		ret = append(ret, head.value)
		next := head.next
		head.next = nil
		head = next
	}

	// Place ambiguous returns after other
	for i := len(ret) - 1; i >= 0; i-- {
		if ret[i].Return == -1 {
			maxTime++
			ret[i].Return = maxTime
		}
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
