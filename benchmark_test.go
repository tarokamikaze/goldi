package goldi

import (
	"reflect"
	"testing"
)

// BenchmarkReflectionCacheSimple tests reflection cache performance
func BenchmarkReflectionCacheSimple(b *testing.B) {
	cache := NewReflectionCache()
	testObj := &struct{ Name string }{Name: "test"}

	b.Run("WithCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.GetType(testObj)
			_ = cache.GetValue(testObj)
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = reflect.TypeOf(testObj)
			_ = reflect.ValueOf(testObj)
		}
	})
}

// BenchmarkMemoryPoolSimple tests memory pool performance
func BenchmarkMemoryPoolSimple(b *testing.B) {
	pool := NewMemoryPool()

	b.Run("WithPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := pool.GetReflectValueSlice()
			slice = append(slice, reflect.ValueOf("test"))
			pool.PutReflectValueSlice(slice)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := make([]reflect.Value, 0, 8)
			slice = append(slice, reflect.ValueOf("test"))
		}
	})
}

// BenchmarkStringSetSimple tests StringSet performance
func BenchmarkStringSetSimple(b *testing.B) {
	set := NewStringSet(100)

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			set.Set("test")
		}
	})

	b.Run("Contains", func(b *testing.B) {
		set.Set("test")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = set.Contains("test")
		}
	})
}
