package goldi_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tarokamikaze/goldi"
	"github.com/tarokamikaze/goldi/validation"
)

// Example_go124Features demonstrates the new Go 1.24 features in Goldi
func Example_go124Features() {
	registry := goldi.NewTypeRegistry()

	// Register some types without dependencies for example
	registry.RegisterType("logger", func() Logger { return &SimpleLogger{} })
	registry.RegisterType("database", func() Database { return &SimpleDatabase{Logger: &SimpleLogger{}} })
	registry.RegisterType("service", func() Service {
		return &SimpleService{DB: &SimpleDatabase{Logger: &SimpleLogger{}}, Logger: &SimpleLogger{}}
	})

	container := goldi.NewContainer(registry, map[string]interface{}{})

	// 1. Range over func - iterate through registered types
	fmt.Println("Registered types:")
	for typeID := range registry.TypeIDs() {
		fmt.Printf("- %s\n", typeID)
	}

	// 2. Improved Type Inference - no type assertions needed
	logger, err := goldi.Get[Logger](container, "logger")
	if err != nil {
		panic(err)
	}
	logger.Log("Using improved type inference!")

	// 3. slices.Collect - efficiently collect type IDs
	allTypeIDs := registry.CollectTypeIDs()
	fmt.Printf("Total registered types: %d\n", len(allTypeIDs))

	// 4. Iterator-based warmup
	err = container.WarmupCache()
	if err != nil {
		panic(err)
	}

	// 5. Check cached instances count
	cachedCount := 0
	for range container.CachedTypeIDs() {
		cachedCount++
	}
	fmt.Printf("Cached instances count: %d\n", cachedCount)

	// Output:
	// Registered types:
	// - logger
	// - database
	// - service
	// LOG: Using improved type inference!
	// Total registered types: 3
	// Cached instances count: 3
}

// Test types for examples
type Logger interface {
	Log(string)
}

type SimpleLogger struct {
	Name string
}

func (l *SimpleLogger) Log(msg string) {
	fmt.Printf("LOG: %s\n", msg)
}

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

func Example() {
	// create a new container when your application loads
	registry := goldi.NewTypeRegistry()
	config := map[string]interface{}{
		"some_parameter": "Hello World",
		"timeout":        42.7,
	}

	// register a simple type with parameter
	registry.RegisterType("logger", NewSimpleLoggerWithParam, "%some_parameter%")

	// register a struct type
	registry.RegisterType("httpClient", &http.Client{}, time.Second*5)

	// you can also register already instantiated types
	registry.InjectInstance("myInstance", &SimpleLogger{Name: "Foo"})

	// create a new container with the registry and the config
	container := goldi.NewContainer(registry, config)

	// retrieve types from the container
	logger := container.MustGet("logger").(*SimpleLogger)
	fmt.Printf("logger.Name = %q\n", logger.Name)

	// Output: logger.Name = "Hello World"
}

func ExampleContainer_RegisterType() {
	registry := goldi.NewTypeRegistry()
	config := map[string]interface{}{
		"some_parameter": "Hello World",
	}

	// register a simple type with parameter
	registry.RegisterType("logger", NewSimpleLoggerWithParam, "%some_parameter%")

	// register a struct type
	registry.RegisterType("httpClient", &http.Client{}, time.Second*5)

	// you can also register already instantiated types
	registry.InjectInstance("myInstance", &SimpleLogger{Name: "Foo"})

	// create a new container with the registry and the config
	container := goldi.NewContainer(registry, config)

	// retrieve types from the container
	logger := container.MustGet("logger").(*SimpleLogger)
	fmt.Printf("logger.Name = %q\n", logger.Name)

	// Output: logger.Name = "Hello World"
}

func ExampleContainer_MustGet() {
	registry := goldi.NewTypeRegistry()
	config := map[string]interface{}{
		"some_parameter": "Hello World",
	}

	registry.RegisterType("logger", NewSimpleLoggerWithParam, "%some_parameter%")
	container := goldi.NewContainer(registry, config)

	logger := container.MustGet("logger").(*SimpleLogger)
	fmt.Printf("logger.Name = %q\n", logger.Name)

	// Output: logger.Name = "Hello World"
}

func ExampleValidateContainer() {
	registry := goldi.NewTypeRegistry()
	config := map[string]interface{}{
		"some_parameter": "Hello World",
	}

	registry.RegisterType("logger", NewSimpleLogger)

	container := goldi.NewContainer(registry, config)

	// this will return an error if the container can not be build successfully
	err := validation.ValidateContainer(container)
	if err != nil {
		fmt.Printf("The container is invalid: %s", err)
		return
	}

	fmt.Println("The container is valid")
	// Output: The container is valid
}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

func NewSimpleLoggerWithParam(name string) *SimpleLogger {
	return &SimpleLogger{Name: name}
}

type MyService struct {
	Logger *SimpleLogger
}

func NewMyService(logger *SimpleLogger) *MyService {
	return &MyService{Logger: logger}
}
