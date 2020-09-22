package tcp

import (
	"fmt"
	"sync"

	"github.com/plgd-dev/go-coap/v2/message"
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
		key = v.String()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.datas[key] != nil {
		return fmt.Errorf("key already exist")
	}
	s.datas[key] = handler
	return nil
}

// Get returns handler for key
func (s *HandlerContainer) Get(key interface{}) (HandlerFunc, error) {
	if v, ok := key.(message.Token); ok {
		key = v.String()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v := s.datas[key]
	if v == nil {
		return nil, fmt.Errorf("key not exist")
	}
	return v, nil
}

// Pop pops handler for key
func (s *HandlerContainer) Pop(key interface{}) (HandlerFunc, error) {
	if v, ok := key.(message.Token); ok {
		key = v.String()
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v := s.datas[key]
	if v == nil {
		return nil, fmt.Errorf("key not exist")
	}
	delete(s.datas, key)
	return v, nil
}
