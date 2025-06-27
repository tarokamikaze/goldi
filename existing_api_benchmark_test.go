package goldi

import (
	"fmt"
	"testing"
)

// BenchmarkExistingAPIUsagePatterns tests the performance of existing APIs with typical user patterns
func BenchmarkExistingAPIUsagePatterns(b *testing.B) {
	// Test the exact pattern mentioned by the user
	b.Run("TypicalUserPattern", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Create registry and register types as user would do
			types := NewTypeRegistry()
			types.RegisterAll(map[string]TypeFactory{
				"elasticsearch.client": NewType(
					func(uri string) string { return "elasticsearch-client-" + uri },
					"%ELASTIC_SEARCH_URI%",
				),
			})

			// Create container and get instance
			container := NewContainer(types, map[string]interface{}{
				"ELASTIC_SEARCH_URI": "http://localhost:9200",
			})

			_, _ = container.Get("elasticsearch.client")
		}
	})

	// Test with pre-created registry (more realistic scenario)
	b.Run("PreCreatedRegistry", func(b *testing.B) {
		types := NewTypeRegistry()
		types.RegisterAll(map[string]TypeFactory{
			"elasticsearch.client": NewType(
				func(uri string) string { return "elasticsearch-client-" + uri },
				"%ELASTIC_SEARCH_URI%",
			),
		})

		container := NewContainer(types, map[string]interface{}{
			"ELASTIC_SEARCH_URI": "http://localhost:9200",
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("elasticsearch.client")
		}
	})

	// Test RegisterAll performance specifically
	b.Run("RegisterAllPerformance", func(b *testing.B) {
		factories := map[string]TypeFactory{
			"service1": NewType(func() string { return "service1" }),
			"service2": NewType(func() string { return "service2" }),
			"service3": NewType(func() string { return "service3" }),
			"service4": NewType(func() string { return "service4" }),
			"service5": NewType(func() string { return "service5" }),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			types := NewTypeRegistry()
			types.RegisterAll(factories)
		}
	})

	// Test NewType performance
	b.Run("NewTypePerformance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewType(
				func(uri string) string { return "client-" + uri },
				"%URI%",
			)
		}
	})

	// Test container.Get performance with caching
	b.Run("ContainerGetWithCaching", func(b *testing.B) {
		types := NewTypeRegistry()
		types.RegisterAll(map[string]TypeFactory{
			"cached.service": NewType(func() string { return "cached-service" }),
		})

		container := NewContainer(types, map[string]interface{}{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("cached.service")
		}
	})

	// Test multiple services pattern
	b.Run("MultipleServicesPattern", func(b *testing.B) {
		types := NewTypeRegistry()
		types.RegisterAll(map[string]TypeFactory{
			"database.client": NewType(
				func(dsn string) string { return "db-client-" + dsn },
				"%DATABASE_DSN%",
			),
			"redis.client": NewType(
				func(host string) string { return "redis-client-" + host },
				"%REDIS_HOST%",
			),
			"logger": NewType(func() string { return "logger" }),
		})

		container := NewContainer(types, map[string]interface{}{
			"DATABASE_DSN": "postgres://localhost:5432/db",
			"REDIS_HOST":   "localhost:6379",
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("database.client")
			_, _ = container.Get("redis.client")
			_, _ = container.Get("logger")
		}
	})
}

// BenchmarkExistingAPIOptimizations tests the effectiveness of current optimizations
func BenchmarkExistingAPIOptimizations(b *testing.B) {
	// Test reflection cache effectiveness
	b.Run("ReflectionCacheEffectiveness", func(b *testing.B) {
		types := NewTypeRegistry()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// This should benefit from reflection caching
			types.RegisterType("service", func() string { return "service" })
		}
	})

	// Test sync.Map vs regular map performance
	b.Run("ThreadSafeCachePerformance", func(b *testing.B) {
		types := NewTypeRegistry()
		types.RegisterType("service", func() string { return "service" })
		container := NewContainer(types, map[string]interface{}{})

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = container.Get("service")
			}
		})
	})

	// Test maps.Copy effectiveness in RegisterAll
	b.Run("MapsCopyEffectiveness", func(b *testing.B) {
		factories := make(map[string]TypeFactory)
		for i := 0; i < 100; i++ {
			factories[fmt.Sprintf("service%d", i)] = NewType(func() string { return "service" })
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			types := NewTypeRegistry()
			types.RegisterAll(factories)
		}
	})
}

// BenchmarkExistingAPIMemoryEfficiency tests memory efficiency improvements
func BenchmarkExistingAPIMemoryEfficiency(b *testing.B) {
	b.Run("MemoryUsageWithOptimizations", func(b *testing.B) {
		b.ReportAllocs()

		types := NewTypeRegistry()
		types.RegisterAll(map[string]TypeFactory{
			"service1": NewType(func() string { return "service1" }),
			"service2": NewType(func() string { return "service2" }),
			"service3": NewType(func() string { return "service3" }),
		})

		container := NewContainer(types, map[string]interface{}{})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("service1")
			_, _ = container.Get("service2")
			_, _ = container.Get("service3")
		}
	})
}
