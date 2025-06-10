//go:build optimize
// +build optimize

package goldi

import (
	"runtime"
	"testing"
)

// BenchmarkOptimizedContainer tests optimized container performance
func BenchmarkOptimizedContainer(b *testing.B) {
	registry := NewTypeRegistry()
	registry.RegisterType("test", func() string { return "test" })

	b.Run("Regular", func(b *testing.B) {
		container := NewContainer(registry, map[string]interface{}{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("test")
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		container := NewOptimizedContainer(registry, map[string]interface{}{})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.FastGet("test")
		}
	})
}

// BenchmarkOptimizedReflectionCache tests optimized reflection cache performance
func BenchmarkOptimizedReflectionCache(b *testing.B) {
	testObj := &struct{ Name string }{Name: "test"}

	b.Run("Regular", func(b *testing.B) {
		cache := NewReflectionCache()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.GetType(testObj)
			_ = cache.GetValue(testObj)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		cache := NewOptimizedReflectionCache()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.FastGetType(testObj)
			_ = cache.FastGetValue(testObj)
		}
	})
}

// BenchmarkOptimizedMemoryPool tests optimized memory pool performance
func BenchmarkOptimizedMemoryPool(b *testing.B) {
	b.Run("Regular", func(b *testing.B) {
		pool := NewMemoryPool()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := pool.GetReflectValueSlice()
			pool.PutReflectValueSlice(slice)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		pool := NewOptimizedMemoryPool()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := pool.GetFastSlice(8)
			pool.PutFastSlice(slice)
		}
	})
}

// TestOptimizedMemoryUsage tests memory usage of optimized components
func TestOptimizedMemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats

	// Measure baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create optimized containers
	containers := make([]*OptimizedContainer, 1000)
	for i := 0; i < 1000; i++ {
		registry := NewTypeRegistry()
		registry.RegisterType("test", func() string { return "test" })
		containers[i] = NewOptimizedContainer(registry, map[string]interface{}{})

		// Generate some instances to test caching
		for j := 0; j < 10; j++ {
			_, _ = containers[i].FastGet("test")
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	t.Logf("Optimized memory allocated: %.2f MB", allocatedMB)
	t.Logf("Total allocations: %d", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("GC cycles: %d", m2.NumGC-m1.NumGC)

	// Keep references to prevent GC
	_ = containers
}

// BenchmarkOptimizedConcurrency tests concurrent performance of optimized components
func BenchmarkOptimizedConcurrency(b *testing.B) {
	registry := NewTypeRegistry()
	registry.RegisterType("test", func() string { return "test" })
	container := NewOptimizedContainer(registry, map[string]interface{}{})

	b.Run("OptimizedConcurrent", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = container.FastGet("test")
			}
		})
	})

	b.Run("RegularConcurrent", func(b *testing.B) {
		regularContainer := NewContainer(registry, map[string]interface{}{})
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = regularContainer.Get("test")
			}
		})
	})
}
