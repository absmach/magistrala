package cache

import (
	"time"

	kitSync "github.com/plgd-dev/kit/v2/sync"
)

func DefaultOnExpire(d interface{}) {
	// for nothing on expire
}

type Element struct {
	validUntil time.Time
	data       interface{}
	onExpire   func(d interface{})
}

func (e *Element) IsExpired(now time.Time) bool {
	if e.validUntil.IsZero() {
		return false
	}
	return now.After(e.validUntil)
}

func (e *Element) Data() interface{} {
	return e.data
}

func NewElement(data interface{}, validUntil time.Time, onExpire func(d interface{})) *Element {
	if onExpire == nil {
		onExpire = DefaultOnExpire
	}
	return &Element{data: data, validUntil: validUntil, onExpire: onExpire}
}

type Cache struct {
	data kitSync.Map
}

func NewCache() *Cache {
	return &Cache{
		data: *kitSync.NewMap(),
	}
}

func (c *Cache) LoadOrStore(key interface{}, e *Element) (actual *Element, loaded bool) {
	now := time.Now()
	c.data.ReplaceWithFunc(key, func(oldValue interface{}, oldLoaded bool) (newValue interface{}, deleteValue bool) {
		if oldLoaded {
			o := oldValue.(*Element)
			if !o.IsExpired(now) {
				actual = o
				return o, false
			}
		}
		actual = e
		return e, false
	})
	return actual, actual != e
}

func (c *Cache) Load(key interface{}) (actual *Element) {
	a, loaded := c.data.Load(key)
	if !loaded {
		return nil
	}
	actual = a.(*Element)
	if actual.IsExpired(time.Now()) {
		return nil
	}
	return actual
}

func (c *Cache) Delete(key interface{}) {
	c.data.Delete(key)
}

func (c *Cache) CheckExpirations(now time.Time) {
	m := make(map[interface{}]*Element)
	c.data.Range(func(key, value interface{}) bool {
		m[key] = value.(*Element)
		return true
	})
	for k, e := range m {
		if e.IsExpired(now) {
			c.data.Delete(k)
			e.onExpire(e.data)
		}
	}
}

func (c *Cache) PullOutAll() map[interface{}]interface{} {
	res := make(map[interface{}]interface{})
	for key, value := range c.data.PullOutAll() {
		res[key] = value.(*Element).Data()
	}
	return res
}
