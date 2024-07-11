package main

import "sync"

type SafeMap struct {
	mut sync.Mutex
	v   map[string]bool
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		v: make(map[string]bool),
	}
}

func (sm *SafeMap) ConditionalInsert(k string) bool {
	sm.mut.Lock()
	defer sm.mut.Unlock()
	if _, exists := sm.v[k]; exists {
		return false
	}
	sm.v[k] = true
	return true
}

func (sm *SafeMap) Insert(k string) {
	sm.mut.Lock()
	sm.v[k] = true
	sm.mut.Unlock()
}
