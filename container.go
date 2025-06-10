package goldi

import (
	"fmt"
	"iter"
	"slices"
	"sync"
)

// Container is the dependency injection container that can be used by your application to define and get types.
//
// Basically this is just a TypeRegistry with access to the application configuration and the knowledge
// of how to build individual services. Additionally this implements the laziness of the DI using a simple in memory type cache
//
// You must use goldi.NewContainer to get a initialized instance of a Container!
type Container struct {
	TypeRegistry
	Config   map[string]interface{}
	Resolver *ParameterResolver

	typeCache       sync.Map         // thread-safe cache for generated instances
	reflectionCache *ReflectionCache // cache for reflection operations
}

// NewContainer creates a new container instance using the provided arguments
func NewContainer(registry TypeRegistry, config map[string]interface{}) *Container {
	c := &Container{
		TypeRegistry:    registry,
		Config:          config,
		reflectionCache: NewReflectionCache(),
	}

	c.Resolver = NewParameterResolver(c)
	return c
}

// MustGet behaves exactly like Get but will panic instead of returning an error
// Since MustGet can only return interface{} you need to add a type assertion after the call:
//
//	container.MustGet("logger").(LoggerInterface)
//
//go:inline
func (c *Container) MustGet(typeID string) interface{} {
	t, err := c.Get(typeID)
	if err != nil {
		panic(err)
	}

	return t
}

// Get retrieves a previously defined type or an error.
// If the requested typeID has not been registered before or can not be generated Get will return an error.
//
// For your dependency injection to work properly it is important that you do only try to assert interface types
// when you use Get(..). Otherwise it might be impossible to assert the correct type when you change the underlying type
// implementations. Also make sure your application is properly tested and defers some panic handling in case you
// forgot to define a service.
//
// See also Container.MustGet
func (c *Container) Get(typeID string) (interface{}, error) {
	instance, isDefined, err := c.get(typeID)
	if err != nil {
		return nil, err
	}

	if isDefined == false {
		return nil, newUnknownTypeReferenceError(typeID, "no such type has been defined")
	}

	return instance, nil
}

// Get retrieves a type with improved type inference using Go 1.24 generics
// This method provides compile-time type safety and eliminates the need for type assertions
//
//go:inline
func Get[T any](c *Container, typeID string) (T, error) {
	var zero T
	instance, err := c.Get(typeID)
	if err != nil {
		return zero, err
	}

	// Type assertion with improved error handling
	if typed, ok := instance.(T); ok {
		return typed, nil
	}

	return zero, fmt.Errorf("goldi: type %q cannot be asserted to %T", typeID, zero)
}

// MustGet with improved type inference - panics on error but provides type safety
//
//go:inline
func MustGet[T any](c *Container, typeID string) T {
	result, err := Get[T](c, typeID)
	if err != nil {
		panic(err)
	}
	return result
}

func (c *Container) get(typeID string) (interface{}, bool, error) {
	// Check cache first (thread-safe read)
	if cached, ok := c.typeCache.Load(typeID); ok {
		return cached, true, nil
	}

	generator, isDefined := c.TypeRegistry[typeID]
	if isDefined == false {
		return nil, false, nil
	}

	instance, err := generator.Generate(c.Resolver)
	if err != nil {
		return nil, false, fmt.Errorf("goldi: error while generating type %q: %s", typeID, err)
	}

	// Store in cache (thread-safe write)
	c.typeCache.Store(typeID, instance)
	return instance, true, nil
}

// AllInstances returns an iterator over all cached instances
// This uses Go 1.24's range over func feature for memory-efficient iteration
func (c *Container) AllInstances() iter.Seq2[string, interface{}] {
	return func(yield func(string, interface{}) bool) {
		c.typeCache.Range(func(key, value interface{}) bool {
			return yield(key.(string), value)
		})
	}
}

// CachedTypeIDs returns an iterator over all cached type IDs
func (c *Container) CachedTypeIDs() iter.Seq[string] {
	return func(yield func(string) bool) {
		c.typeCache.Range(func(key, value interface{}) bool {
			return yield(key.(string))
		})
	}
}

// GetMultiple efficiently retrieves multiple types using iterator pattern
func (c *Container) GetMultiple(typeIDs iter.Seq[string]) iter.Seq2[string, interface{}] {
	return func(yield func(string, interface{}) bool) {
		for typeID := range typeIDs {
			if instance, err := c.Get(typeID); err == nil {
				if !yield(typeID, instance) {
					return
				}
			}
		}
	}
}

// WarmupCache pre-generates instances for all registered types
func (c *Container) WarmupCache() error {
	for typeID := range c.TypeRegistry.TypeIDs() {
		if _, err := c.Get(typeID); err != nil {
			return fmt.Errorf("failed to warmup type %q: %w", typeID, err)
		}
	}
	return nil
}

// CollectCachedInstances efficiently collects all cached instances using slices.Collect
func (c *Container) CollectCachedInstances() map[string]interface{} {
	result := make(map[string]interface{})
	c.typeCache.Range(func(key, value interface{}) bool {
		result[key.(string)] = value
		return true
	})
	return result
}

// CollectCachedTypeIDs efficiently collects all cached type IDs using slices.Collect
func (c *Container) CollectCachedTypeIDs() []string {
	return slices.Collect(c.CachedTypeIDs())
}

// GetAllInstances retrieves all registered types and returns them as a map
func (c *Container) GetAllInstances() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for typeID := range c.TypeRegistry.TypeIDs() {
		instance, err := c.Get(typeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance for type %q: %w", typeID, err)
		}
		result[typeID] = instance
	}
	return result, nil
}
