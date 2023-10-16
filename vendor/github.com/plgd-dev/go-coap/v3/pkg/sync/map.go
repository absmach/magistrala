package sync

import (
	"sync"

	"golang.org/x/exp/maps"
)

// Map is like a Go map[interface{}]interface{} but is safe for concurrent use by multiple goroutines.
type Map[K comparable, V any] struct {
	mutex sync.RWMutex
	data  map[K]V
}

// NewMap creates map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data: make(map[K]V),
	}
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = value
}

// Load returns the value stored in the map for a key, or nil if no value is present. The loaded value is read-only and should not be modified.
// The ok result indicates whether value was found in the map.
func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

// LoadOrStore returns the existing value for the key if present. The loaded value is read-only and should not be modified.
// Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	m.mutex.RLock()
	v, ok := m.data[key]
	m.mutex.RUnlock()
	if ok {
		return v, true
	}
	m.mutex.Lock()
	m.data[key] = value
	m.mutex.Unlock()
	return value, false
}

// Replace replaces the existing value with a new value and returns old value for the key.
func (m *Map[K, V]) Replace(key K, value V) (oldValue V, oldLoaded bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.data[key]
	m.data[key] = value
	return v, ok
}

// Delete deletes the value for the key.
func (m *Map[K, V]) Delete(key K) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.data, key)
}

// LoadAndDelete loads and deletes the value for the key.
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	value, ok := m.data[key]
	delete(m.data, key)
	return value, ok
}

// LoadAndDelete loads and deletes the value for the key.
func (m *Map[K, V]) LoadAndDeleteAll() map[K]V {
	m.mutex.Lock()
	data := m.data
	m.data = make(map[K]V)
	m.mutex.Unlock()
	return data
}

// CopyData creates a deep copy of the internal map.
func (m *Map[K, V]) CopyData() map[K]V {
	c := make(map[K]V)
	m.mutex.RLock()
	maps.Copy(c, m.data)
	m.mutex.RUnlock()
	return c
}

// Length returns number of stored values.
func (m *Map[K, V]) Length() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.data)
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
//
// Range does not copy the whole map, instead the read lock is locked on iteration of the map, and unlocked before f is called.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for key, value := range m.data {
		m.mutex.RUnlock()
		ok := f(key, value)
		m.mutex.RLock()
		if !ok {
			return
		}
	}
}

// Range2 calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
//
// Range2 differs from Range by keepting a read lock locked during the whole call.
func (m *Map[K, V]) Range2(f func(key K, value V) bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for key, value := range m.data {
		ok := f(key, value)
		if !ok {
			return
		}
	}
}

// StoreWithFunc creates a new element and stores it in the map under the given key.
//
// The createFunc is invoked under a write lock.
func (m *Map[K, V]) StoreWithFunc(key K, createFunc func() V) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = createFunc()
}

// LoadWithFunc tries to load element for key from the map, if it exists then the onload functions is invoked on it.
//
// The onLoadFunc is invoked under a read lock.
func (m *Map[K, V]) LoadWithFunc(key K, onLoadFunc func(value V) V) (V, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	value, ok := m.data[key]
	if ok && onLoadFunc != nil {
		value = onLoadFunc(value)
	}
	return value, ok
}

// LoadOrStoreWithFunc loads an existing element from the map or creates a new element and stores it in the map
//
// The onLoadFunc or createFunc are invoked under a write lock.
func (m *Map[K, V]) LoadOrStoreWithFunc(key K, onLoadFunc func(value V) V, createFunc func() V) (actual V, loaded bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.data[key]
	if ok {
		if onLoadFunc != nil {
			v = onLoadFunc(v)
		}
		return v, true
	}
	v = createFunc()
	m.data[key] = v
	return v, false
}

// ReplaceWithFunc checks whether key exists in the map, invokes the onReplaceFunc callback on the pair (value, ok) and either deletes or stores the element
// in the map based on the returned values from the onReplaceFunc callback.
//
// The onReplaceFunc callback is invoked under a write lock.
func (m *Map[K, V]) ReplaceWithFunc(key K, onReplaceFunc func(oldValue V, oldLoaded bool) (newValue V, doDelete bool)) (oldValue V, oldLoaded bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.data[key]
	newValue, del := onReplaceFunc(v, ok)
	if del {
		delete(m.data, key)
		return v, ok
	}
	m.data[key] = newValue
	return v, ok
}

// DeleteWithFunc removes the key from the map and if a value existed invokes the onDeleteFunc callback on the removed value.
//
// The onDeleteFunc callback is invoked under a write lock.
func (m *Map[K, V]) DeleteWithFunc(key K, onDeleteFunc func(value V)) {
	_, _ = m.LoadAndDeleteWithFunc(key, func(value V) V {
		onDeleteFunc(value)
		return value
	})
}

// LoadAndDeleteWithFunc removes the key from the map and if a value existed invokes the onLoadFunc callback on the removed and return it.
//
// The onLoadFunc callback is invoked under a write lock.
func (m *Map[K, V]) LoadAndDeleteWithFunc(key K, onLoadFunc func(value V) V) (V, bool) {
	var v V
	var loaded bool
	m.ReplaceWithFunc(key, func(oldValue V, oldLoaded bool) (newValue V, doDelete bool) {
		if oldLoaded {
			loaded = true
			v = onLoadFunc(oldValue)
			return v, true
		}
		return oldValue, true
	})
	return v, loaded
}
