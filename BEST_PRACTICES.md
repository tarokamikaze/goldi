# Goldi Best Practices Guide

This guide provides recommendations for optimal usage of Goldi dependency injection framework, especially leveraging Go 1.24 features.

## 🎯 Type Registration Best Practices

### Use Type-Safe Generics
```go
// ✅ Recommended: Type-safe retrieval
logger, err := goldi.Get[Logger](container, "logger")
if err != nil {
    return err
}

// ❌ Avoid: Manual type assertions
loggerInterface, err := container.Get("logger")
if err != nil {
    return err
}
logger := loggerInterface.(Logger) // Runtime panic risk
```

### Prefer Factory Functions Over Structs
```go
// ✅ Recommended: Factory functions for complex initialization
registry.RegisterType("database", func() Database {
    return &PostgresDB{
        Host: "localhost",
        Port: 5432,
        Pool: createConnectionPool(),
    }
})

// ❌ Avoid: Direct struct registration for complex types
registry.RegisterType("database", &PostgresDB{}) // Missing initialization
```

## 🚀 Performance Optimization

### Leverage Warm Cache for Frequently Used Services
```go
// Warm up cache for critical services at startup
criticalServices := []string{"logger", "database", "cache", "auth"}
for _, serviceID := range criticalServices {
    _, _ = container.Get(serviceID) // Pre-populate cache
}

// Or use bulk warmup
err := container.WarmupCache()
if err != nil {
    log.Fatal("Failed to warmup cache:", err)
}
```

### Use Memory Pools for High-Frequency Operations
```go
// ✅ Recommended: Use memory pools for repeated allocations
pool := goldi.NewMemoryPool()

func processRequests(requests []Request) {
    slice := pool.GetReflectValueSlice()
    defer pool.PutReflectValueSlice(slice)
    
    // Process requests using pooled slice
    for _, req := range requests {
        slice = append(slice, reflect.ValueOf(req))
    }
}
```

### Optimize Iterator Usage
```go
// ✅ Recommended: Use iterators for memory efficiency
count := 0
for range registry.TypeIDs() {
    count++
}

// ❌ Avoid: Collecting all items when only counting
allIDs := registry.CollectTypeIDs() // Unnecessary allocation
count := len(allIDs)
```

## 🔒 Concurrency Best Practices

### Container Thread Safety
```go
// ✅ Safe: Containers are thread-safe for reads
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        logger, _ := goldi.Get[Logger](container, "logger")
        logger.Log("Concurrent access is safe")
    }()
}
wg.Wait()
```

### Avoid Registration During Runtime
```go
// ✅ Recommended: Register all types at startup
func initializeContainer() *goldi.Container {
    registry := goldi.NewTypeRegistry()
    
    // Register all types here
    registry.RegisterType("logger", func() Logger { return &FileLogger{} })
    registry.RegisterType("database", func() Database { return &PostgresDB{} })
    
    return goldi.NewContainer(registry, config)
}

// ❌ Avoid: Runtime registration in concurrent environment
func handleRequest(container *goldi.Container) {
    // This is not thread-safe and should be avoided
    container.RegisterType("temp_service", &TempService{})
}
```

## 🏗️ Architecture Patterns

### Service Layer Pattern
```go
// Define service interfaces
type UserService interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}

type EmailService interface {
    SendEmail(to, subject, body string) error
}

// Register implementations
registry.RegisterType("user_service", func() UserService {
    return &userServiceImpl{
        db: goldi.MustGet[Database](container, "database"),
        logger: goldi.MustGet[Logger](container, "logger"),
    }
})

registry.RegisterType("email_service", func() EmailService {
    return &emailServiceImpl{
        smtp: goldi.MustGet[SMTPClient](container, "smtp_client"),
    }
})
```

### Configuration Management
```go
// ✅ Recommended: Centralized configuration
type AppConfig struct {
    Database DatabaseConfig `json:"database"`
    Redis    RedisConfig    `json:"redis"`
    SMTP     SMTPConfig     `json:"smtp"`
}

registry.RegisterType("config", func() *AppConfig {
    return loadConfigFromFile("config.json")
})

registry.RegisterType("database", func() Database {
    config := goldi.MustGet[*AppConfig](container, "config")
    return NewDatabase(config.Database)
})
```

## 🧪 Testing Best Practices

### Use Test Containers
```go
func TestUserService(t *testing.T) {
    // Create test-specific container
    registry := goldi.NewTypeRegistry()
    
    // Register test doubles
    registry.RegisterType("database", func() Database {
        return &MockDatabase{} // Test implementation
    })
    
    registry.RegisterType("user_service", func() UserService {
        return &userServiceImpl{
            db: goldi.MustGet[Database](container, "database"),
        }
    })
    
    container := goldi.NewContainer(registry, map[string]interface{}{})
    
    // Test with clean container
    service := goldi.MustGet[UserService](container, "user_service")
    user, err := service.GetUser("123")
    
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

### Mock Dependencies
```go
// ✅ Recommended: Interface-based mocking
type MockLogger struct {
    messages []string
}

func (m *MockLogger) Log(msg string) {
    m.messages = append(m.messages, msg)
}

func TestWithMockLogger(t *testing.T) {
    mockLogger := &MockLogger{}
    
    registry := goldi.NewTypeRegistry()
    registry.RegisterType("logger", func() Logger { return mockLogger })
    
    container := goldi.NewContainer(registry, map[string]interface{}{})
    
    // Test code here
    logger := goldi.MustGet[Logger](container, "logger")
    logger.Log("test message")
    
    assert.Contains(t, mockLogger.messages, "test message")
}
```

## 📊 Monitoring and Debugging

### Performance Monitoring
```go
// Monitor container performance
func monitorContainerPerformance(container *goldi.Container) {
    start := time.Now()
    
    // Measure cache hit rate
    cachedTypes := container.CollectCachedTypeIDs()
    allTypes := container.CollectTypeIDs()
    
    hitRate := float64(len(cachedTypes)) / float64(len(allTypes)) * 100
    
    log.Printf("Container stats: %.1f%% cache hit rate, %d total types, took %v",
        hitRate, len(allTypes), time.Since(start))
}
```

### Debug Type Resolution
```go
// Debug type resolution issues
func debugTypeResolution(container *goldi.Container, typeID string) {
    log.Printf("Attempting to resolve type: %s", typeID)
    
    // Check if type is registered
    if factory := container.TypeRegistry[typeID]; factory == nil {
        log.Printf("ERROR: Type %s is not registered", typeID)
        return
    }
    
    // Check if type is cached
    if cached := container.CollectCachedTypeIDs(); slices.Contains(cached, typeID) {
        log.Printf("Type %s is cached", typeID)
    } else {
        log.Printf("Type %s will be generated fresh", typeID)
    }
    
    // Attempt resolution
    start := time.Now()
    instance, err := container.Get(typeID)
    duration := time.Since(start)
    
    if err != nil {
        log.Printf("ERROR resolving %s: %v (took %v)", typeID, err, duration)
    } else {
        log.Printf("Successfully resolved %s to %T (took %v)", typeID, instance, duration)
    }
}
```

## 🚨 Common Pitfalls to Avoid

### 1. Circular Dependencies
```go
// ❌ Avoid: Circular dependencies
registry.RegisterType("service_a", func() ServiceA {
    return &serviceAImpl{
        serviceB: goldi.MustGet[ServiceB](container, "service_b"),
    }
})

registry.RegisterType("service_b", func() ServiceB {
    return &serviceBImpl{
        serviceA: goldi.MustGet[ServiceA](container, "service_a"), // Circular!
    }
})
```

### 2. Heavy Initialization in Factory Functions
```go
// ❌ Avoid: Heavy operations in factory functions
registry.RegisterType("database", func() Database {
    // This blocks container initialization
    time.Sleep(5 * time.Second) // Heavy initialization
    return &PostgresDB{}
})

// ✅ Recommended: Lazy initialization
registry.RegisterType("database", func() Database {
    return &LazyPostgresDB{} // Initialize on first use
})
```

### 3. Ignoring Error Handling
```go
// ❌ Avoid: Ignoring errors
logger := goldi.MustGet[Logger](container, "logger") // Panics on error

// ✅ Recommended: Proper error handling
logger, err := goldi.Get[Logger](container, "logger")
if err != nil {
    return fmt.Errorf("failed to get logger: %w", err)
}
```

## 📚 Additional Resources

- [Go 1.24 Release Notes](https://golang.org/doc/go1.24)
- [Goldi API Documentation](https://godoc.org/github.com/fgrosse/goldi)
- [Dependency Injection Patterns](https://martinfowler.com/articles/injection.html)
- [Go Concurrency Patterns](https://golang.org/doc/effective_go.html#concurrency)