package goldi

import (
	"fmt"
	"reflect"
)

// A structType holds all information that is necessary to create a new instance of some struct type.
// structType implements the TypeFactory interface.
type structType struct {
	structType   reflect.Type
	structFields []reflect.Value
}

// NewStructType creates a TypeFactory that can be used to create a new instance of some struct type.
//
// This function will return an invalid type if:
//   - structT is no struct or pointer to a struct,
//   - the number of given structParameters exceed the number of field of structT
//   - the structParameters types do not match the fields of structT
//
// Goldigen yaml syntax example:
//
//	logger:
//	    package: github.com/fgrosse/foobar
//	    type:    MyType
func NewStructType(structT interface{}, structParameters ...interface{}) TypeFactory {
	if structT == nil {
		return newInvalidType(fmt.Errorf("the given struct is nil"))
	}

	structType := reflect.TypeOf(structT)
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	switch structType.Kind() {
	case reflect.Struct:
		return newTypeFromStruct(structType, structParameters)
	default:
		return newInvalidType(fmt.Errorf("the given type must either be a struct or a pointer to a struct (given %T)", structT))
	}
}

func newTypeFromStruct(generatedType reflect.Type, parameters []interface{}) TypeFactory {
	if generatedType.NumField() < len(parameters) {
		return newInvalidType(fmt.Errorf("the struct %s has only %d fields but %d arguments where provided",
			generatedType.Name(), generatedType.NumField(), len(parameters),
		))
	}

	// Use cached reflection operations
	cache := GetGlobalReflectionCache()
	args := make([]reflect.Value, len(parameters))
	for i, argument := range parameters {
		// TODO: check argument types
		args[i] = cache.GetValue(argument)
	}

	return &structType{
		structType:   generatedType,
		structFields: args,
	}
}

// Arguments returns all struct parameters from NewStructType
func (t *structType) Arguments() []interface{} {
	args := make([]interface{}, len(t.structFields))
	for i, argument := range t.structFields {
		args[i] = argument.Interface()
	}
	return args
}

// Generate will instantiate a new instance of the according type.
func (t *structType) Generate(parameterResolver *ParameterResolver) (interface{}, error) {
	args, err := t.generateTypeFields(parameterResolver)
	if err != nil {
		return nil, err
	}

	newStructInstance := reflect.New(t.structType)
	for i := 0; i < len(args); i++ {
		newStructInstance.Elem().Field(i).Set(args[i])
	}

	return newStructInstance.Interface(), nil
}

func (t *structType) generateTypeFields(parameterResolver *ParameterResolver) ([]reflect.Value, error) {
	// Pre-allocate with known size for better performance
	args := make([]reflect.Value, len(t.structFields))
	var err error

	for i, argument := range t.structFields {
		expectedArgument := t.structType.Field(i).Type
		args[i], err = parameterResolver.Resolve(argument, expectedArgument)

		switch errorType := err.(type) {
		case nil:
			continue
		case TypeReferenceError:
			return nil, t.invalidReferencedTypeErr(errorType.TypeID, errorType.TypeInstance, i)
		default:
			return nil, err
		}
	}

	return args, nil
}

func (t *structType) invalidReferencedTypeErr(typeID string, typeInstance interface{}, i int) error {
	err := fmt.Errorf("the referenced type \"@%s\" (type %T) can not be used as field %d for struct type %v",
		typeID, typeInstance, i+1, t.structType,
	)

	return err
}
