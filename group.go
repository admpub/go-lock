package lock

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Group ...
type Group interface {
	Lock(key interface{})
	TryLock(key interface{}) bool
	TryLockWithTimeout(key interface{}, duration time.Duration) bool
	TryLockWithContext(key interface{}, ctx context.Context) bool
	Unlock(key interface{})
	UnlockAndFree(key interface{})
}

func NewGroup(fn func() Mutex) Group {
	if fn == nil {
		fn = func() Mutex {
			return NewChanMutex()
		}
	}
	return &group{
		fn:    fn,
		group: make(map[interface{}]*entry),
	}
}

type group struct {
	mu    sync.RWMutex
	fn    func() Mutex
	group map[interface{}]*entry
}

type entry struct {
	ref int32
	mu  Mutex
}

func (m *group) get(i interface{}, ref int32) Mutex {
	m.mu.RLock()
	en, ok := m.group[i]
	m.mu.RUnlock()
	if !ok {
		if ref > 0 {
			en = &entry{mu: m.fn()}
			m.mu.Lock()
			m.group[i] = en
			m.mu.Unlock()
		} else {
			return nil
		}
	}
	atomic.AddInt32(&en.ref, ref)
	return en.mu
}

func (m *group) Lock(i interface{}) {
	m.get(i, 1).Lock()
}

func (m *group) Unlock(i interface{}) {
	mu := m.get(i, -1)
	if mu != nil {
		mu.Unlock()
	}
}

func (m *group) UnlockAndFree(i interface{}) {
	m.mu.RLock()
	en, ok := m.group[i]
	m.mu.RUnlock()
	if !ok {
		return
	}
	atomic.AddInt32(&en.ref, -1)
	if atomic.LoadInt32(&en.ref) <= 0 {
		m.mu.Lock()
		delete(m.group, i)
		m.mu.Unlock()
	}
	en.mu.Unlock()
}

func (m *group) TryLock(i interface{}) bool {
	locked := m.get(i, 1).TryLock()
	if !locked {
		m.get(i, -1)
	}
	return locked
}

func (m *group) TryLockWithTimeout(i interface{}, timeout time.Duration) bool {
	locked := m.get(i, 1).TryLockWithTimeout(timeout)
	if !locked {
		m.get(i, -1)
	}
	return locked
}

func (m *group) TryLockWithContext(i interface{}, ctx context.Context) bool {
	locked := m.get(i, 1).TryLockWithContext(ctx)
	if !locked {
		m.get(i, -1)
	}
	return locked
}
