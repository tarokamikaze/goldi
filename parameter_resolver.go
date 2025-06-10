package goldi

import "reflect"

// The ParameterResolver is used by type factories to resolve the values of the dynamic factory arguments
// (parameters and other type references).
type ParameterResolver struct {
	Container *Container
}

// NewParameterResolver creates a new ParameterResolver and initializes it with the given Container.
// The container is used when resolving parameters and the type references.
func NewParameterResolver(container *Container) *ParameterResolver {
	return &ParameterResolver{
		Container: container,
	}
}

// Resolve takes a parameter and resolves any references to configuration parameter values or type references.
// If the type of `parameter` is not a parameter or type reference it is returned as is.
// Parameters must always have the form `%my.beautiful.param%.
// Type references must have the form `@my_type.bla`.
// It is also legal to request an optional type using the syntax `@?my_optional_type`.
// If this type is not registered Resolve will not return an error but instead give you the null value
// of the expected type.
func (r *ParameterResolver) Resolve(parameter reflect.Value, expectedType reflect.Type) (reflect.Value, error) {
	if parameter.Kind() != reflect.String {
		return parameter, nil
	}

	stringParameter := parameter.Interface().(string)
	if IsParameterOrTypeReference(stringParameter) == false {
		return parameter, nil
	}

	if IsTypeReference(stringParameter) {
		return r.resolveTypeReference(stringParameter, expectedType)
	}

	return r.resolveParameter(parameter, stringParameter, expectedType), nil
}

func (r *ParameterResolver) resolveParameter(parameter reflect.Value, stringParameter string, expectedType reflect.Type) reflect.Value {
	parameterName := stringParameter[1 : len(stringParameter)-1]
	configuredValue, isConfigured := r.Container.Config[parameterName]
	if isConfigured == false {
		return parameter
	}

	// Use cached reflection operations
	cache := GetGlobalReflectionCache()
	parameter = reflect.New(expectedType).Elem()
	parameter.Set(cache.GetValue(configuredValue))
	return parameter
}

func (r *ParameterResolver) resolveTypeReference(typeIDAndPrefix string, expectedType reflect.Type) (reflect.Value, error) {
	t := NewTypeID(typeIDAndPrefix)

	typeInstance, typeDefined, err := r.Container.get(t.ID)
	if err != nil {
		return reflect.Zero(expectedType), err
	}

	if typeDefined == false {
		if t.IsOptional {
			return reflect.Zero(expectedType), nil
		}

		return reflect.Value{}, newUnknownTypeReferenceError(t.ID, `the referenced type "@%s" has not been defined`, t.ID)
	}

	if t.IsFuncReference {
		// Use cached reflection operations
		cache := GetGlobalReflectionCache()
		objValue := cache.GetValue(typeInstance)
		method := objValue.MethodByName(t.FuncReferenceMethod)

		if method.IsValid() == false {
			return reflect.Value{}, newTypeReferenceError(t.ID, typeInstance, `the referenced method %q does not exist or is not exported`, t.Raw)
		}

		return method, nil
	}

	// Use cached reflection operations
	cache := GetGlobalReflectionCache()
	instanceType := cache.GetType(typeInstance)
	if instanceType.AssignableTo(expectedType) == false {
		return reflect.Value{}, newTypeReferenceError(t.ID, typeInstance,
			`the referenced type %q (type %T) is not assignable to the expected type %v`, t.Raw, typeInstance, expectedType,
		)
	}

	result := reflect.New(expectedType).Elem()
	result.Set(cache.GetValue(typeInstance))
	return result, nil
}
