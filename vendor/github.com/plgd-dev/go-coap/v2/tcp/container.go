package tcp

import (
	"errors"
	"sync"

	"github.com/plgd-dev/go-coap/v2/message"
)

var (
	ErrKeyAlreadyExists = errors.New("key already exists")

	ErrKeyNotExists = errors.New("key does not exist")
)

// HandlerContainer for regirstration handlers by key
type HandlerContainer struct {
	datas map[interface{}]HandlerFunc
	mutex sync.Mutex
}

// NewHandlerContainer factory
func NewHandlerContainer() *HandlerContainer {
	return &HandlerContainer{
		datas: make(map[interface{}]HandlerFunc),
	}
}

// Insert handler for key.
func (s *HandlerContainer) Insert(key interface{}, handler HandlerFunc) error {
	if v, ok := key.(message.Token); ok {
		key = v.Hash()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, keyExists := s.datas[key]; keyExists {
		return ErrKeyAlreadyExists
	}
	s.datas[key] = handler
	return nil
}

// Get returns handler for key
func (s *HandlerContainer) Get(key interface{}) (HandlerFunc, error) {
	if v, ok := key.(message.Token); ok {
		key = v.Hash()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v, keyExists := s.datas[key]
	if !keyExists {
		return nil, ErrKeyNotExists
	}
	return v, nil
}

// Pop pops handler for key
func (s *HandlerContainer) Pop(key interface{}) (HandlerFunc, error) {
	if v, ok := key.(message.Token); ok {
		key = v.Hash()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v, keyExists := s.datas[key]
	if !keyExists {
		return nil, ErrKeyNotExists
	}
	delete(s.datas, key)
	return v, nil
}
