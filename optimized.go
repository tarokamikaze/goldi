//go:build optimize
// +build optimize

package goldi

import (
	"reflect"
	"sync"
	"unsafe"
)

// OptimizedContainer provides compile-time optimized version of Container
type OptimizedContainer struct {
	*Container
	fastCache map[string]unsafe.Pointer
	cacheMu   sync.RWMutex
}

// NewOptimizedContainer creates a new optimized container
func NewOptimizedContainer(registry TypeRegistry, config map[string]interface{}) *OptimizedContainer {
	return &OptimizedContainer{
		Container: NewContainer(registry, config),
		fastCache: make(map[string]unsafe.Pointer),
	}
}

// FastGet provides optimized instance retrieval using unsafe pointers
func (c *OptimizedContainer) FastGet(typeID string) (interface{}, error) {
	c.cacheMu.RLock()
	if ptr, exists := c.fastCache[typeID]; exists {
		c.cacheMu.RUnlock()
		return *(*interface{})(ptr), nil
	}
	c.cacheMu.RUnlock()

	// Fallback to regular Get and cache result
	instance, err := c.Container.Get(typeID)
	if err != nil {
		return nil, err
	}

	c.cacheMu.Lock()
	c.fastCache[typeID] = unsafe.Pointer(&instance)
	c.cacheMu.Unlock()

	return instance, nil
}

// OptimizedReflectionCache provides compile-time optimized reflection caching
type OptimizedReflectionCache struct {
	*ReflectionCache
	typeCache  map[uintptr]reflect.Type
	valueCache map[uintptr]reflect.Value
	cacheMu    sync.RWMutex
}

// NewOptimizedReflectionCache creates a new optimized reflection cache
func NewOptimizedReflectionCache() *OptimizedReflectionCache {
	return &OptimizedReflectionCache{
		ReflectionCache: NewReflectionCache(),
		typeCache:       make(map[uintptr]reflect.Type),
		valueCache:      make(map[uintptr]reflect.Value),
	}
}

// FastGetType provides optimized type retrieval using pointer addresses
func (c *OptimizedReflectionCache) FastGetType(obj interface{}) reflect.Type {
	ptr := uintptr(unsafe.Pointer(&obj))

	c.cacheMu.RLock()
	if typ, exists := c.typeCache[ptr]; exists {
		c.cacheMu.RUnlock()
		return typ
	}
	c.cacheMu.RUnlock()

	typ := reflect.TypeOf(obj)

	c.cacheMu.Lock()
	c.typeCache[ptr] = typ
	c.cacheMu.Unlock()

	return typ
}

// FastGetValue provides optimized value retrieval using pointer addresses
func (c *OptimizedReflectionCache) FastGetValue(obj interface{}) reflect.Value {
	ptr := uintptr(unsafe.Pointer(&obj))

	c.cacheMu.RLock()
	if val, exists := c.valueCache[ptr]; exists {
		c.cacheMu.RUnlock()
		return val
	}
	c.cacheMu.RUnlock()

	val := reflect.ValueOf(obj)

	c.cacheMu.Lock()
	c.valueCache[ptr] = val
	c.cacheMu.Unlock()

	return val
}

// OptimizedMemoryPool provides compile-time optimized memory pooling
type OptimizedMemoryPool struct {
	*MemoryPool
	fastPools map[int]*sync.Pool
	poolMu    sync.RWMutex
}

// NewOptimizedMemoryPool creates a new optimized memory pool
func NewOptimizedMemoryPool() *OptimizedMemoryPool {
	return &OptimizedMemoryPool{
		MemoryPool: NewMemoryPool(),
		fastPools:  make(map[int]*sync.Pool),
	}
}

// GetFastSlice provides optimized slice allocation for specific sizes
func (p *OptimizedMemoryPool) GetFastSlice(size int) []reflect.Value {
	p.poolMu.RLock()
	pool, exists := p.fastPools[size]
	p.poolMu.RUnlock()

	if !exists {
		p.poolMu.Lock()
		if pool, exists = p.fastPools[size]; !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					return make([]reflect.Value, 0, size)
				},
			}
			p.fastPools[size] = pool
		}
		p.poolMu.Unlock()
	}

	return pool.Get().([]reflect.Value)
}

// PutFastSlice returns optimized slice to the appropriate pool
func (p *OptimizedMemoryPool) PutFastSlice(slice []reflect.Value) {
	size := cap(slice)

	p.poolMu.RLock()
	pool, exists := p.fastPools[size]
	p.poolMu.RUnlock()

	if exists {
		slice = slice[:0] // Reset length but keep capacity
		pool.Put(slice)
	}
}

// Compile-time optimization hints
//
//go:noinline
func init() {
	// Force initialization of optimized components
	_ = NewOptimizedContainer(nil, nil)
	_ = NewOptimizedReflectionCache()
	_ = NewOptimizedMemoryPool()
}
