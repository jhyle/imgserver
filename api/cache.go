package imgserver

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

type (
	CacheStats struct {
		Items  uint32
		Size   uint64
		Gets   uint64
		Puts   uint64
		Hits   uint64
		Prunes uint64
	}

	ByteCache interface {
		FindKeys(prefix string) []string
		Put(key string, value []byte, mod time.Time) []byte
		Get(key string, mod time.Time) []byte
		Remove(keys []string) [][]byte
		Stats() CacheStats
	}

	mapCache struct {
		sync.RWMutex
		keys  []string
		mods  map[string]time.Time
		m     map[string][]byte
		max   uint64
		stats CacheStats
	}
)

func NewByteCache(capacity uint64) ByteCache {

	return &mapCache{m: make(map[string][]byte), mods: make(map[string]time.Time), max: capacity}
}

func (cache *mapCache) removeKey(i int) {

	v := cache.m[cache.keys[i]]
	if v != nil {
		cache.stats.Size -= uint64(len(v))
	}
	cache.stats.Items--

	delete(cache.m, cache.keys[i])
	copy(cache.keys[i:], cache.keys[i+1:])
	cache.keys = cache.keys[:len(cache.keys)-1]
}

func (cache *mapCache) Put(key string, value []byte, mod time.Time) []byte {

	if uint64(len(value)) > cache.max {
		return nil
	}
	cache.Lock()

	neededCapacity := len(value)
	oldValue, exists := cache.m[key]
	if oldValue != nil {
		neededCapacity -= len(oldValue)
	}

	for cache.stats.Size+uint64(neededCapacity) > cache.max {
		i := rand.Intn(len(cache.keys))
		if cache.keys[i] != key {
			cache.removeKey(i)
			cache.stats.Prunes++
		}
	}

	if !exists {
		cache.keys = append(cache.keys, key)
		cache.stats.Items++
	}

	cache.m[key] = value
	cache.stats.Size += uint64(neededCapacity)

	cache.mods[key] = mod;
	cache.Unlock()
	cache.stats.Puts++
	return oldValue
}

func (cache *mapCache) Get(key string, mod time.Time) []byte {

	var item []byte = nil;
	cache.stats.Gets++
	cache.RLock()
	
	cachedMod := cache.mods[key]
	if cachedMod.Equal(mod) {
		item = cache.m[key]
		if item != nil {
			cache.stats.Hits++
		} 
	}
	
	cache.RUnlock()
	return item
}

func (cache *mapCache) Remove(keys []string) [][]byte {

	cache.Lock()
	var oldValues = make([][]byte, len(keys), len(keys))

	for n := 0; n < len(keys); n++ {
		key := keys[n]
		if v, exists := cache.m[key]; exists {
			oldValues[n] = v
			for i := 0; i < len(cache.keys); i++ {
				if cache.keys[i] == key {
					cache.removeKey(i)
					break
				}
			}
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

	return cache.stats
}
