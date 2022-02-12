package sync

import "sync"

// Map is like a Go map[interface{}]interface{} but is safe for concurrent use by multiple goroutines.
type Map struct {
	mutex sync.Mutex
	data  map[interface{}]interface{}
}

// NewMap creates map.
func NewMap() *Map {
	return &Map{
		data: make(map[interface{}]interface{}),
	}
}

// Delete deletes the value for a key.
func (m *Map) Delete(key interface{}) {
	m.DeleteWithFunc(key, nil)
}

func (m *Map) DeleteWithFunc(key interface{}, onDeleteFunc func(value interface{})) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	value, ok := m.data[key]
	delete(m.data, key)
	if ok && onDeleteFunc != nil {
		onDeleteFunc(value)
	}
}

// Load returns the value stored in the map for a key, or nil if no value is present.
// The ok result indicates whether value was found in the map.
func (m *Map) Load(key interface{}) (value interface{}, ok bool) {
	return m.LoadWithFunc(key, nil)
}

func (m *Map) LoadWithFunc(key interface{}, onLoadFunc func(value interface{}) interface{}) (value interface{}, ok bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	value, ok = m.data[key]
	if ok && onLoadFunc != nil {
		value = onLoadFunc(value)
	}
	return
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) {
	return m.LoadOrStoreWithFunc(key, nil, func() interface{} { return value })
}

func (m *Map) LoadOrStoreWithFunc(key interface{}, onLoadFunc func(value interface{}) interface{}, createFunc func() interface{}) (actual interface{}, loaded bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.data[key]
	if ok {
		if onLoadFunc != nil {
			v = onLoadFunc(v)
		}
		return v, ok
	}
	v = createFunc()
	m.data[key] = v
	return v, ok
}

// Replace replaces the existing value with a new value and returns old value for the key.
func (m *Map) Replace(key, value interface{}) (oldValue interface{}, oldLoaded bool) {
	return m.ReplaceWithFunc(key, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool) { return value, false })
}

func (m *Map) ReplaceWithFunc(key interface{}, onReplaceFunc func(oldValue interface{}, oldLoaded bool) (newValue interface{}, delete bool)) (oldValue interface{}, oldLoaded bool) {
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

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
func (m *Map) Range(f func(key, value interface{}) bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for key, value := range m.data {
		ok := f(key, value)
		if !ok {
			return
		}
	}
}

// Store sets the value for a key.
func (m *Map) Store(key, value interface{}) {
	m.StoreWithFunc(key, func() interface{} { return value })
}

func (m *Map) StoreWithFunc(key interface{}, createFunc func() interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.data[key] = createFunc()
}

// PullOut loads and deletes the value for a key.
func (m *Map) PullOut(key interface{}) (value interface{}, ok bool) {
	return m.PullOutWithFunc(key, nil)
}

func (m *Map) PullOutWithFunc(key interface{}, onLoadFunc func(value interface{}) interface{}) (value interface{}, ok bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	value, ok = m.data[key]
	delete(m.data, key)
	if ok && onLoadFunc != nil {
		value = onLoadFunc(value)
	}
	return
}

// PullOutAll extracts internal map data and replace it with empty map.
func (m *Map) PullOutAll() (data map[interface{}]interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	data = m.data
	m.data = make(map[interface{}]interface{})
	return
}

// Length returns number of stored values.
func (m *Map) Length() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.data)
}
