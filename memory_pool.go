package goldi

import (
	"reflect"
	"sync"
)

// MemoryPool provides object pooling to reduce memory allocations
type MemoryPool struct {
	reflectValuePool sync.Pool
	slicePool        sync.Pool
	stringPool       sync.Pool
}

// NewMemoryPool creates a new memory pool instance
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{
		reflectValuePool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate slice with reasonable capacity
				return make([]reflect.Value, 0, 8)
			},
		},
		slicePool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate interface slice with reasonable capacity
				return make([]interface{}, 0, 8)
			},
		},
		stringPool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate string slice with reasonable capacity
				return make([]string, 0, 8)
			},
		},
	}
}

// GetReflectValueSlice returns a pooled reflect.Value slice
//
//go:inline
func (mp *MemoryPool) GetReflectValueSlice() []reflect.Value {
	slice := mp.reflectValuePool.Get().([]reflect.Value)
	return slice[:0] // Reset length but keep capacity
}

// PutReflectValueSlice returns a reflect.Value slice to the pool
//
//go:inline
func (mp *MemoryPool) PutReflectValueSlice(slice []reflect.Value) {
	if cap(slice) > 64 { // Prevent memory leaks from very large slices
		return
	}
	mp.reflectValuePool.Put(slice)
}

// GetInterfaceSlice returns a pooled interface{} slice
func (mp *MemoryPool) GetInterfaceSlice() []interface{} {
	slice := mp.slicePool.Get().([]interface{})
	return slice[:0] // Reset length but keep capacity
}

// PutInterfaceSlice returns an interface{} slice to the pool
func (mp *MemoryPool) PutInterfaceSlice(slice []interface{}) {
	if cap(slice) > 64 { // Prevent memory leaks from very large slices
		return
	}
	mp.slicePool.Put(slice)
}

// GetStringSlice returns a pooled string slice
func (mp *MemoryPool) GetStringSlice() []string {
	slice := mp.stringPool.Get().([]string)
	return slice[:0] // Reset length but keep capacity
}

// PutStringSlice returns a string slice to the pool
func (mp *MemoryPool) PutStringSlice(slice []string) {
	if cap(slice) > 64 { // Prevent memory leaks from very large slices
		return
	}
	mp.stringPool.Put(slice)
}

// Global memory pool instance
var globalMemoryPool = NewMemoryPool()

// GetGlobalMemoryPool returns the global memory pool instance
func GetGlobalMemoryPool() *MemoryPool {
	return globalMemoryPool
}
