package imgserver

import (
	"math/rand"
	"strings"
	"sync"
)

type (
	CacheStats struct {
		Items int
		Gets  int
		Puts  int
		Hits  int
	}

	Cache interface {
		FindKeys(prefix string) []string
		Put(key string, value interface{}) interface{}
		Get(key string) interface{}
		Remove(keys []string) []interface{}
		Stats() CacheStats
	}

	mapCache struct {
		sync.RWMutex
		keys []string
		m    map[string]interface{}
		stats CacheStats
	}
)

func NewMapCache(capacity int) Cache {
	return &mapCache{keys: make([]string, 0, capacity), m: make(map[string]interface{}, capacity)}
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
			cache.stats.Items++
		}
	}

	cache.m[key] = value
	cache.stats.Puts++

	cache.Unlock()
	return oldValue
}

func (cache *mapCache) Get(key string) interface{} {

	cache.RLock()

	cache.stats.Gets++
	item := cache.m[key]
	if item != nil {
		cache.stats.Hits++
	}

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
			cache.stats.Items--

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

	for i := 0; i < len(cache.keys); i++ {
		if strings.HasPrefix(cache.keys[i], prefix) {
			keys = append(keys, cache.keys[i])
		}
	}

	cache.RUnlock()
	return keys
}

func (cache *mapCache) Stats() CacheStats {

	return cache.stats;
}