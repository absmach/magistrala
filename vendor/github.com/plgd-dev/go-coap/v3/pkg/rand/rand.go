package rand

import (
	"math/rand"
	"sync"
)

type Rand struct {
	src  *rand.Rand
	lock sync.Mutex
}

func NewRand(seed int64) *Rand {
	return &Rand{
		src: rand.New(rand.NewSource(seed)),
	}
}

func (l *Rand) Int63() int64 {
	l.lock.Lock()
	val := l.src.Int63()
	l.lock.Unlock()
	return val
}

func (l *Rand) Uint32() uint32 {
	l.lock.Lock()
	val := l.src.Uint32()
	l.lock.Unlock()
	return val
}
