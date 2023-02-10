package utils

import (
	"sync"
	"time"
)

var CLEANUP_INTERVAL = 1 * time.Minute

type forgetfulMapInner[V comparable] struct {
	inner      V
	accessTime time.Time
}

type ForgetfulMap[K comparable, V comparable] struct {
	Mutex    *sync.RWMutex
	TTL      time.Duration
	innerMap map[K]forgetfulMapInner[V]
}

func NewForgetfulMap[K comparable, V comparable](TTL time.Duration) ForgetfulMap[K, V] {
	fm := ForgetfulMap[K, V]{&sync.RWMutex{}, TTL, make(map[K]forgetfulMapInner[V])}

	go func() {
		for range time.NewTicker(CLEANUP_INTERVAL).C {
			fm.cleanup()
		}
	}()

	return fm
}

func (fm *ForgetfulMap[K, V]) Get(key K) (V, bool) {
	fm.Mutex.RLock()
	inner, exists := fm.innerMap[key]
	fm.Mutex.RUnlock()
	if exists {
		fm.Mutex.Lock()
		inner.accessTime = time.Now()
		fm.innerMap[key] = inner
		fm.Mutex.Unlock()
	}
	return inner.inner, exists
}

func (fm *ForgetfulMap[K, V]) Set(key K, value V) {
	fm.Mutex.Lock()
	fm.innerMap[key] = forgetfulMapInner[V]{inner: value, accessTime: time.Now()}
	fm.Mutex.Unlock()
}

func (fm *ForgetfulMap[K, V]) Delete(key K) {
	fm.Mutex.Lock()
	delete(fm.innerMap, key)
	fm.Mutex.Unlock()
}

func (fm *ForgetfulMap[K, V]) cleanup() {
	fm.Mutex.Lock()
	for k, inner := range fm.innerMap {
		if time.Since(inner.accessTime) <= fm.TTL {
			continue
		}

		delete(fm.innerMap, k)
	}
	fm.Mutex.Unlock()
}
