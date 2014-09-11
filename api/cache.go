package imgserver

import (
	"math/rand"
	"sync"
	"strings"
)

type (
	Cache interface {
		FindKeys(prefix string) []string
		Put(key string, value interface{}) interface{}
		Get(key string) interface{}
		Remove(keys []string) []interface{}
	}

	mapCache struct {
		sync.RWMutex
		keys []string
		m    map[string]interface{}
	}
)

func NewMapCache(capacity int) Cache {
	return &mapCache{keys: make([]string, 0, capacity), m: make(map[string]interface{})}
}

func (cache *mapCache) Put(key string, value interface{}) interface{} {

	cache.Lock()
	var oldValue interface{} = nil

	if v, exists := cache.m[key]; exists {
		oldValue = v
	} else {
		if len(cache.keys) == cap(cache.keys) {
			i := rand.Intn(len(cache.keys))
			delete(cache.m, cache.keys[i])
			cache.keys[i] = key
		} else {
			cache.keys = append(cache.keys, key)
		}
	}

	cache.m[key] = value
	cache.Unlock()

	return oldValue
}

func (cache *mapCache) Get(key string) interface{} {

	cache.RLock()
	item := cache.m[key]
	cache.RUnlock()

	return item
}

func (cache *mapCache) Remove(keys []string) []interface{} {

	cache.Lock()
	var oldValues = make([]interface{}, len(keys), len(keys))

	for n := 0; n < len(keys); n++ {
		key := keys[n]
		if v, exists := cache.m[key]; exists {
			oldValues[n] = v
			delete(cache.m, key)

			var i int
			for i = 0; i < len(cache.keys); i++ {
				if cache.keys[i] == key {
					break
				}
			}

			copy(cache.keys[i:], cache.keys[i+1:])
			cache.keys = cache.keys[:len(cache.keys)-1]
		}
	}

	cache.Unlock()
	return oldValues
}

func (cache *mapCache) FindKeys(prefix string) []string {

	cache.RLock()
	keys := make([]string, 0, 4)

	for i:= 0; i < len(cache.keys); i++ {
		if strings.HasPrefix(cache.keys[i], prefix) {
			keys = append(keys, cache.keys[i])
		}
	}
	
	cache.RUnlock()
	return keys
}
