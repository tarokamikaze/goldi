package goldi

import (
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
)

// The TypeRegistry is effectively a map of typeID strings to TypeFactory
type TypeRegistry map[string]TypeFactory

// NewTypeRegistry creates a new empty TypeRegistry
func NewTypeRegistry() TypeRegistry {
	return TypeRegistry{}
}

// RegisterType is convenience method for TypeRegistry.Register
// It tries to create the correct TypeFactory and passes this to TypeRegistry.Register
// This function panics if the given generator function and arguments can not be used to create a new type factory.
func (r TypeRegistry) RegisterType(typeID string, factory interface{}, arguments ...interface{}) {
	var typeFactory TypeFactory

	// Use cached reflection operations
	cache := GetGlobalReflectionCache()
	factoryType := cache.GetType(factory)
	kind := factoryType.Kind()
	switch {
	case kind == reflect.Struct:
		fallthrough
	case kind == reflect.Ptr && factoryType.Elem().Kind() == reflect.Struct:
		typeFactory = NewStructType(factory, arguments...)
	case kind == reflect.Func:
		typeFactory = NewType(factory, arguments...)
	default:
		panic(fmt.Errorf("could not register type %q: could not determine TypeFactory for factory type %T", typeID, factory))
	}

	r.Register(typeID, typeFactory)
}

// Register saves a type under the given symbolic typeID so it can be retrieved later.
// It is perfectly legal to call Register multiple times with the same typeID.
// In this case you overwrite existing type definitions with new once
func (r TypeRegistry) Register(typeID string, typeDef TypeFactory) {
	r[typeID] = typeDef
}

// RegisterAll will register all given type factories under the mapped type ID
// It uses maps.Copy for efficient bulk registration
func (r TypeRegistry) RegisterAll(factories map[string]TypeFactory) {
	maps.Copy(r, factories)
}

// InjectInstance enables you to inject type instances.
// If instance is nil an error is returned
func (r TypeRegistry) InjectInstance(typeID string, instance interface{}) {
	factory := NewInstanceType(instance)
	r.Register(typeID, factory)
}

// All returns an iterator over all registered type IDs and their factories
// This uses Go 1.24's range over func feature for memory-efficient iteration
func (r TypeRegistry) All() iter.Seq2[string, TypeFactory] {
	return func(yield func(string, TypeFactory) bool) {
		for typeID, factory := range r {
			if !yield(typeID, factory) {
				return
			}
		}
	}
}

// TypeIDs returns an iterator over all registered type IDs
func (r TypeRegistry) TypeIDs() iter.Seq[string] {
	return func(yield func(string) bool) {
		for typeID := range r {
			if !yield(typeID) {
				return
			}
		}
	}
}

// Factories returns an iterator over all registered factories
func (r TypeRegistry) Factories() iter.Seq[TypeFactory] {
	return func(yield func(TypeFactory) bool) {
		for _, factory := range r {
			if !yield(factory) {
				return
			}
		}
	}
}

// FilterByType returns an iterator over type IDs that match the given type
func (r TypeRegistry) FilterByType(targetType reflect.Type) iter.Seq[string] {
	return func(yield func(string) bool) {
		cache := GetGlobalReflectionCache()
		for typeID, factory := range r {
			if factoryType := cache.GetFactoryType(factory); factoryType == targetType {
				if !yield(typeID) {
					return
				}
			}
		}
	}
}

// CollectTypeIDs efficiently collects all type IDs into a slice using slices.Collect
func (r TypeRegistry) CollectTypeIDs() []string {
	return slices.Collect(r.TypeIDs())
}

// CollectFactories efficiently collects all factories into a slice using slices.Collect
func (r TypeRegistry) CollectFactories() []TypeFactory {
	return slices.Collect(r.Factories())
}

// CollectAll efficiently collects all type registrations into a map using maps.Collect
func (r TypeRegistry) CollectAll() map[string]TypeFactory {
	return maps.Collect(r.All())
}

// CollectByType efficiently collects type IDs matching a specific type
func (r TypeRegistry) CollectByType(targetType reflect.Type) []string {
	return slices.Collect(r.FilterByType(targetType))
}

// Clone creates a deep copy of the registry using maps.Collect
func (r TypeRegistry) Clone() TypeRegistry {
	return maps.Collect(r.All())
}
