package client

import (
	"fmt"
	"sync"
)

// MutexMap wraps a map of mutexes.  Each key locks separately.
type MutexMap struct {
	ma map[interface{}]*mutexMapEntry // entry map
	ml sync.Mutex                     // lock for entry map
}

type mutexMapEntry struct {
	key interface{} // key in ma
	m   *MutexMap   // point back to MutexMap, so we can synchronize removing this mutexMapEntry when cnt==0
	el  sync.Mutex  // entry-specific lock
	cnt uint16      // reference count
}

// Unlocker provides an Unlock method to release the lock.
type Unlocker interface {
	Unlock()
}

// NewMutexMap returns an initialized MutexMap.
func NewMutexMap() *MutexMap {
	return &MutexMap{ma: make(map[interface{}]*mutexMapEntry)}
}

// Lock acquires a lock corresponding to this key.
// This method will never return nil and Unlock() must be called
// to release the lock when done.
func (m *MutexMap) Lock(key interface{}) Unlocker {
	// read or create entry for this key atomically
	m.ml.Lock()
	e, ok := m.ma[key]
	if !ok {
		e = &mutexMapEntry{m: m, key: key}
		m.ma[key] = e
	}
	e.cnt++ // ref count
	m.ml.Unlock()

	// acquire lock, will block here until e.cnt==1
	e.el.Lock()

	return e
}

// Unlock releases the lock for this entry.
func (entry *mutexMapEntry) Unlock() {
	m := entry.m

	// decrement and if needed remove entry atomically
	m.ml.Lock()
	e, ok := m.ma[entry.key]
	if !ok { // entry must exist
		m.ml.Unlock()
		panic(fmt.Errorf("unlock requested for key=%v but no entry found", entry.key))
	}
	e.cnt--        // ref count
	if e.cnt < 1 { // if it hits zero then we own it and remove from map
		delete(m.ma, entry.key)
	}
	m.ml.Unlock()

	// now that map stuff is handled, we unlock and let
	// anything else waiting on this key through
	e.el.Unlock()
}
