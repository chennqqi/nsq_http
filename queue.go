package main

import (
	"container/list"
	"fmt"
	"sync"
)

type Queue struct {
	list   *list.List
	mutext sync.Mutex
	ch     chan byte
}

func NewQueue(length int) *Queue {
	var q Queue
	l := list.New()
	l.Init()
	q.list = l
	q.ch = make(chan byte, length)
	return &q
}

func (q *Queue) Len() int {
	q.mutext.Lock()
	defer q.mutext.Unlock()
	l := q.list
	return l.Len()
}

func (q *Queue) PopFront() interface{} {
	fmt.Println("POPFRONT")
	<-q.ch
	q.mutext.Lock()
	defer q.mutext.Unlock()
	l := q.list
	f := l.Front()
	if f != nil {
		return l.Remove(f)
	}
	return nil
}

func (q *Queue) PopBack() interface{} {
	fmt.Println("POPBACK")
	<-q.ch
	q.mutext.Lock()
	defer q.mutext.Unlock()
	l := q.list
	f := l.Back()
	if f != nil {
		return l.Remove(f)
	}
	return nil
}

func (q *Queue) PushFront(element interface{}) {
	fmt.Println("PUSHFRONT")
	q.mutext.Lock()
	defer q.mutext.Unlock()
	l := q.list
	l.PushFront(element)
	q.ch <- '0'
}

func (q *Queue) PushBack(element interface{}) {
	fmt.Println("PUSHBACK")
	q.mutext.Lock()
	defer q.mutext.Unlock()
	l := q.list
	l.PushBack(element)
	q.ch <- '0'
}
