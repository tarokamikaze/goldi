package goldi

import (
	"reflect"
	"sync"
)

// ReflectionCache provides cached reflection operations to reduce runtime overhead
type ReflectionCache struct {
	typeCache   sync.Map // map[string]reflect.Type (using type string as key)
	valueCache  sync.Map // map[string]reflect.Value (using type string as key)
	methodCache sync.Map // map[string]reflect.Method
}

// NewReflectionCache creates a new reflection cache instance
func NewReflectionCache() *ReflectionCache {
	return &ReflectionCache{}
}

// GetType returns cached reflect.Type or computes and caches it
//
//go:inline
func (rc *ReflectionCache) GetType(obj interface{}) reflect.Type {
	// Use type string as key to avoid unhashable types
	typeKey := getTypeKey(obj)
	if cached, ok := rc.typeCache.Load(typeKey); ok {
		return cached.(reflect.Type)
	}

	reflectType := reflect.TypeOf(obj)
	rc.typeCache.Store(typeKey, reflectType)
	return reflectType
}

// GetValue returns cached reflect.Value or computes and caches it
// Note: Values are not cached as they represent specific instances
//
//go:inline
func (rc *ReflectionCache) GetValue(obj interface{}) reflect.Value {
	// Don't cache values as they represent specific instances
	// Only cache types for performance optimization
	return reflect.ValueOf(obj)
}

// GetMethodByName returns cached method or computes and caches it
func (rc *ReflectionCache) GetMethodByName(obj interface{}, methodName string) (reflect.Method, bool) {
	key := getMethodCacheKey(obj, methodName)
	if cached, ok := rc.methodCache.Load(key); ok {
		method := cached.(methodCacheEntry)
		return method.Method, method.Valid
	}

	objType := rc.GetType(obj)
	method, valid := objType.MethodByName(methodName)

	entry := methodCacheEntry{Method: method, Valid: valid}
	rc.methodCache.Store(key, entry)

	return method, valid
}

// GetFactoryType returns the type that a TypeFactory produces
func (rc *ReflectionCache) GetFactoryType(factory TypeFactory) reflect.Type {
	// Get the factory's underlying function type
	factoryType := rc.GetType(factory)

	// For TypeFactory interface, we need to get the Generate method's return type
	if method, ok := rc.GetMethodByName(factory, "Generate"); ok {
		methodType := method.Type
		if methodType.NumOut() >= 1 {
			return methodType.Out(0) // Return type of Generate method
		}
	}

	// Fallback: return the factory type itself
	return factoryType
}

// getTypeKey generates a unique string key for any type
func getTypeKey(obj interface{}) string {
	objType := reflect.TypeOf(obj)
	return objType.String()
}

// methodCacheEntry stores method lookup results
type methodCacheEntry struct {
	Method reflect.Method
	Valid  bool
}

// getMethodCacheKey generates a unique key for method caching
func getMethodCacheKey(obj interface{}, methodName string) string {
	return getTypeKey(obj) + "::" + methodName
}

// Global reflection cache instance
var globalReflectionCache = NewReflectionCache()

// GetGlobalReflectionCache returns the global reflection cache instance
func GetGlobalReflectionCache() *ReflectionCache {
	return globalReflectionCache
}
