package goldi

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkReflectionCache tests the performance improvement of reflection caching
func BenchmarkReflectionCache(b *testing.B) {
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

// BenchmarkContainerConcurrency tests concurrent access to container
func BenchmarkContainerConcurrency(b *testing.B) {
	registry := NewTypeRegistry()
	registry.RegisterType("test", func() string { return "test" })
	container := NewContainer(registry, map[string]interface{}{})

	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("test")
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = container.Get("test")
			}
		})
	})
}

// BenchmarkMemoryPool tests memory pool performance
func BenchmarkMemoryPool(b *testing.B) {
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

// BenchmarkStringSet tests StringSet performance
func BenchmarkStringSet(b *testing.B) {
	set := NewStringSet(100)

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			set.Set(fmt.Sprintf("key%d", i%100))
		}
	})

	b.Run("Contains", func(b *testing.B) {
		// Pre-populate
		for i := 0; i < 100; i++ {
			set.Set(fmt.Sprintf("key%d", i))
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = set.Contains(fmt.Sprintf("key%d", i%100))
		}
	})
}

// TestMemoryUsage tests memory usage improvements
func TestMemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats

	// Measure baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create many containers to test memory efficiency
	containers := make([]*Container, 1000)
	for i := 0; i < 1000; i++ {
		registry := NewTypeRegistry()
		registry.RegisterType("test", func() string { return "test" })
		containers[i] = NewContainer(registry, map[string]interface{}{})

		// Generate some instances to test caching
		for j := 0; j < 10; j++ {
			_, _ = containers[i].Get("test")
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
	t.Logf("Memory allocated: %.2f MB", allocatedMB)
	t.Logf("Total allocations: %d", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("GC cycles: %d", m2.NumGC-m1.NumGC)

	// Keep references to prevent GC
	_ = containers
}

// TestConcurrentSafety tests thread safety of optimized components
func TestConcurrentSafety(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterType("test", func() string { return "test" })
	container := NewContainer(registry, map[string]interface{}{})

	const numGoroutines = 100
	const numOperations = 1000

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Test concurrent container access
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_, err := container.Get("test")
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

// BenchmarkComplexTypeGeneration tests performance with complex type hierarchies
func BenchmarkComplexTypeGeneration(b *testing.B) {
	registry := NewTypeRegistry()

	// Register simple types without dependencies for benchmarking
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	registry.RegisterType("database", func() Database {
		return &SimpleDatabase{Logger: &SimpleLogger{}}
	})
	registry.RegisterType("service", func() Service {
		return &SimpleService{DB: &SimpleDatabase{Logger: &SimpleLogger{}}, Logger: &SimpleLogger{}}
	})

	container := NewContainer(registry, map[string]interface{}{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := container.Get("service")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test types for benchmarking
type Logger interface {
	Log(string)
}

type SimpleLogger struct{}

func (l *SimpleLogger) Log(msg string) {}

type Database interface {
	Query(string) string
}

type SimpleDatabase struct {
	Logger Logger
}

func (d *SimpleDatabase) Query(query string) string {
	d.Logger.Log("Executing query: " + query)
	return "result"
}

type Service interface {
	Process(string) string
}

type SimpleService struct {
	DB     Database
	Logger Logger
}

func (s *SimpleService) Process(data string) string {
	s.Logger.Log("Processing: " + data)
	return s.DB.Query("SELECT * FROM data WHERE value = '" + data + "'")
}

// PerformanceProfiler provides detailed performance metrics
type PerformanceProfiler struct {
	startTime time.Time
	metrics   map[string]time.Duration
}

func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		startTime: time.Now(),
		metrics:   make(map[string]time.Duration),
	}
}

func (p *PerformanceProfiler) Mark(name string) {
	p.metrics[name] = time.Since(p.startTime)
}

func (p *PerformanceProfiler) Report() map[string]time.Duration {
	return p.metrics
}

// BenchmarkGo124Features tests the performance improvements from Go 1.24 features
func BenchmarkGo124Features(b *testing.B) {
	registry := NewTypeRegistry()
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	registry.RegisterType("database", func(logger Logger) Database { return &SimpleDatabase{Logger: logger} })
	registry.RegisterType("service", func(db Database, logger Logger) Service {
		return &SimpleService{DB: db, Logger: logger}
	})
	container := NewContainer(registry, map[string]interface{}{})

	b.Run("IteratorVsTraditional", func(b *testing.B) {
		b.Run("WithIterator", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				count := 0
				for range registry.TypeIDs() {
					count++
				}
				_ = count
			}
		})

		b.Run("TraditionalLoop", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				count := 0
				for range registry {
					count++
				}
				_ = count
			}
		})
	})

	b.Run("SlicesCollectVsManual", func(b *testing.B) {
		b.Run("WithSlicesCollect", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = registry.CollectTypeIDs()
			}
		})

		b.Run("ManualCollection", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var ids []string
				for id := range registry {
					ids = append(ids, id)
				}
				_ = ids
			}
		})
	})

	b.Run("GenericGetVsTypeAssertion", func(b *testing.B) {
		// Warmup
		_, _ = container.Get("logger")

		b.Run("WithGenerics", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = Get[Logger](container, "logger")
			}
		})

		b.Run("WithTypeAssertion", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				instance, _ := container.Get("logger")
				_ = instance.(Logger)
			}
		})
	})
}

// BenchmarkIteratorMemoryEfficiency tests memory efficiency of iterators
func BenchmarkIteratorMemoryEfficiency(b *testing.B) {
	registry := NewTypeRegistry()

	// Register many types
	for i := 0; i < 1000; i++ {
		typeID := fmt.Sprintf("type_%d", i)
		registry.RegisterType(typeID, func() string { return "test" })
	}

	b.Run("IteratorMemory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range registry.TypeIDs() {
				count++
			}
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})

	b.Run("CollectionMemory", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.CollectTypeIDs()
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	})
}

// BenchmarkContainerIterators tests container iterator performance
func BenchmarkContainerIterators(b *testing.B) {
	registry := NewTypeRegistry()
	for i := 0; i < 100; i++ {
		typeID := fmt.Sprintf("type_%d", i)
		registry.RegisterType(typeID, func() string { return "test" })
	}
	container := NewContainer(registry, map[string]interface{}{})

	// Warmup cache
	_ = container.WarmupCache()

	b.Run("AllInstancesIterator", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range container.AllInstances() {
				count++
			}
		}
	})

	b.Run("CachedTypeIDsIterator", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range container.CachedTypeIDs() {
				count++
			}
		}
	})

	b.Run("GetMultiple", func(b *testing.B) {
		typeIDs := registry.CollectTypeIDs()[:10] // First 10 types
		// Create iterator from slice
		typeIDsIter := func(yield func(string) bool) {
			for _, typeID := range typeIDs {
				if !yield(typeID) {
					return
				}
			}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range container.GetMultiple(typeIDsIter) {
				count++
			}
		}
	})
}

// TestGo124FeatureIntegration tests integration of all Go 1.24 features
func TestGo124FeatureIntegration(t *testing.T) {
	registry := NewTypeRegistry()
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	// Use simple types without dependencies for testing
	registry.RegisterType("service1", func() string { return "service1" })
	registry.RegisterType("service2", func() string { return "service2" })
	container := NewContainer(registry, map[string]interface{}{})

	// Test iterator functionality
	t.Run("IteratorFunctionality", func(t *testing.T) {
		typeIDs := registry.CollectTypeIDs()
		if len(typeIDs) != 3 {
			t.Errorf("Expected 3 type IDs, got %d", len(typeIDs))
		}

		// Test iterator over all registrations
		count := 0
		for typeID, factory := range registry.All() {
			if factory == nil {
				t.Errorf("Factory for %s is nil", typeID)
			}
			count++
		}
		if count != 3 {
			t.Errorf("Expected 3 iterations, got %d", count)
		}
	})

	// Test improved type inference
	t.Run("ImprovedTypeInference", func(t *testing.T) {
		logger, err := Get[Logger](container, "logger")
		if err != nil {
			t.Fatalf("Failed to get logger with generics: %v", err)
		}
		if logger == nil {
			t.Error("Logger is nil")
		}

		// Test with simple string type
		service1, err := Get[string](container, "service1")
		if err != nil {
			t.Fatalf("Failed to get service1 with generics: %v", err)
		}
		if service1 != "service1" {
			t.Errorf("Expected 'service1', got %s", service1)
		}
	})

	// Test slices/maps collection
	t.Run("SlicesMapsCollection", func(t *testing.T) {
		allTypes := registry.CollectAll()
		if len(allTypes) != 3 {
			t.Errorf("Expected 3 collected types, got %d", len(allTypes))
		}

		clonedRegistry := registry.Clone()
		if len(clonedRegistry) != len(registry) {
			t.Error("Cloned registry has different size")
		}
	})

	// Test container warmup and caching
	t.Run("ContainerWarmupAndCaching", func(t *testing.T) {
		err := container.WarmupCache()
		if err != nil {
			t.Fatalf("Failed to warmup cache: %v", err)
		}

		cachedIDs := container.CollectCachedTypeIDs()
		if len(cachedIDs) != 3 {
			t.Errorf("Expected 3 cached types, got %d", len(cachedIDs))
		}

		allInstances, err := container.GetAllInstances()
		if err != nil {
			t.Fatalf("Failed to get all instances: %v", err)
		}
		if len(allInstances) != 3 {
			t.Errorf("Expected 3 instances, got %d", len(allInstances))
		}
	})
}

// BenchmarkRealWorldUsagePatterns tests performance with realistic usage scenarios
func BenchmarkRealWorldUsagePatterns(b *testing.B) {
	registry := NewTypeRegistry()

	// Simulate a web application with multiple services
	registry.RegisterType("config", func() map[string]string {
		return map[string]string{"env": "production", "debug": "false"}
	})
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	registry.RegisterType("database", func() Database {
		return &SimpleDatabase{Logger: &SimpleLogger{}}
	})
	registry.RegisterType("cache", func() string { return "redis-cache" })
	registry.RegisterType("auth_service", func() string { return "jwt-auth" })
	registry.RegisterType("user_service", func() Service {
		return &SimpleService{DB: &SimpleDatabase{Logger: &SimpleLogger{}}, Logger: &SimpleLogger{}}
	})
	registry.RegisterType("api_handler", func() string { return "rest-api" })

	container := NewContainer(registry, map[string]interface{}{})

	b.Run("ColdStart", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate cold start - create new container each time
			newContainer := NewContainer(registry, map[string]interface{}{})
			_, _ = newContainer.Get("user_service")
			_, _ = newContainer.Get("api_handler")
			_, _ = newContainer.Get("auth_service")
		}
	})

	b.Run("WarmCache", func(b *testing.B) {
		// Warmup
		_, _ = container.Get("user_service")
		_, _ = container.Get("api_handler")
		_, _ = container.Get("auth_service")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = container.Get("user_service")
			_, _ = container.Get("api_handler")
			_, _ = container.Get("auth_service")
		}
	})

	b.Run("MixedAccess", func(b *testing.B) {
		services := []string{"config", "logger", "database", "cache", "auth_service", "user_service", "api_handler"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			serviceID := services[i%len(services)]
			_, _ = container.Get(serviceID)
		}
	})
}

// BenchmarkMemoryEfficiencyComparison compares memory efficiency of different approaches
func BenchmarkMemoryEfficiencyComparison(b *testing.B) {
	registry := NewTypeRegistry()
	for i := 0; i < 50; i++ {
		typeID := fmt.Sprintf("service_%d", i)
		registry.RegisterType(typeID, func() string { return "service" })
	}

	b.Run("WithMemoryPool", func(b *testing.B) {
		pool := NewMemoryPool()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := pool.GetReflectValueSlice()
			for j := 0; j < 10; j++ {
				slice = append(slice, reflect.ValueOf("test"))
			}
			pool.PutReflectValueSlice(slice)
		}
	})

	b.Run("WithoutMemoryPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slice := make([]reflect.Value, 0, 8)
			for j := 0; j < 10; j++ {
				slice = append(slice, reflect.ValueOf("test"))
			}
		}
	})
}

// BenchmarkConcurrentRealWorld tests concurrent access patterns in real-world scenarios
func BenchmarkConcurrentRealWorld(b *testing.B) {
	registry := NewTypeRegistry()
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	registry.RegisterType("database", func() Database {
		return &SimpleDatabase{Logger: &SimpleLogger{}}
	})
	registry.RegisterType("user_service", func() Service {
		return &SimpleService{DB: &SimpleDatabase{Logger: &SimpleLogger{}}, Logger: &SimpleLogger{}}
	})

	container := NewContainer(registry, map[string]interface{}{})

	b.Run("HighConcurrency", func(b *testing.B) {
		b.SetParallelism(100) // High concurrency
		b.RunParallel(func(pb *testing.PB) {
			services := []string{"logger", "database", "user_service"}
			i := 0
			for pb.Next() {
				serviceID := services[i%len(services)]
				_, _ = container.Get(serviceID)
				i++
			}
		})
	})

	b.Run("MediumConcurrency", func(b *testing.B) {
		b.SetParallelism(10) // Medium concurrency
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = container.Get("user_service")
			}
		})
	})
}

// BenchmarkTypeRegistryOperations tests type registry performance
func BenchmarkTypeRegistryOperations(b *testing.B) {
	b.Run("Registration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			registry := NewTypeRegistry()
			for j := 0; j < 100; j++ {
				typeID := fmt.Sprintf("service_%d", j)
				registry.RegisterType(typeID, func() string { return "service" })
			}
		}
	})

	b.Run("Lookup", func(b *testing.B) {
		registry := NewTypeRegistry()
		for i := 0; i < 100; i++ {
			typeID := fmt.Sprintf("service_%d", i)
			registry.RegisterType(typeID, func() string { return "service" })
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			typeID := fmt.Sprintf("service_%d", i%100)
			_ = registry[typeID] // Direct map access
		}
	})

	b.Run("IteratorPerformance", func(b *testing.B) {
		registry := NewTypeRegistry()
		for i := 0; i < 1000; i++ {
			typeID := fmt.Sprintf("service_%d", i)
			registry.RegisterType(typeID, func() string { return "service" })
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for range registry.TypeIDs() {
				count++
			}
		}
	})
}
