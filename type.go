package goldi

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// A Type holds all information that is necessary to create a new instance of a type.
// Type implements the TypeFactory interface.
type typeFactory struct {
	factory          reflect.Value
	factoryType      reflect.Type
	factoryArguments []reflect.Value
}

// NewType creates a new TypeFactory.
//
// This function will return an invalid type if:
//   - the factoryFunction is nil or no function,
//   - the factoryFunction returns zero or more than one parameter
//   - the factoryFunctions return parameter is no pointer, interface  or function type.
//   - the number of given factoryParameters does not match the number of arguments of the factoryFunction
//
// Goldigen yaml syntax example:
//
//	my_type:
//	    package: github.com/fgrosse/foobar
//	    factory: NewType
//	    args:
//	        - "Hello World"
//	        - true
func NewType(factoryFunction interface{}, factoryParameters ...interface{}) TypeFactory {
	if factoryFunction == nil {
		return newInvalidType(fmt.Errorf("the given factoryFunction is nil"))
	}

	factoryType := reflect.TypeOf(factoryFunction)
	kind := factoryType.Kind()
	switch {
	case kind == reflect.Func:
		return newTypeFromFactoryFunction(factoryFunction, factoryType, factoryParameters)
	default:
		return newInvalidType(fmt.Errorf("the given factoryFunction must be a function (given %q)", factoryType.Kind()))
	}
}

func newTypeFromFactoryFunction(function interface{}, factoryType reflect.Type, parameters []interface{}) TypeFactory {
	if factoryType.NumOut() != 1 {
		return newInvalidType(fmt.Errorf("invalid number of return parameters: %d", factoryType.NumOut()))
	}

	kindOfGeneratedType := factoryType.Out(0).Kind()
	// Allow more return types including basic types like string, int, etc.
	// Only restrict channels and unsafe pointers which are problematic for DI
	if kindOfGeneratedType == reflect.Chan || kindOfGeneratedType == reflect.UnsafePointer {
		return newInvalidType(fmt.Errorf("return parameter type %v is not supported for dependency injection", kindOfGeneratedType))
	}

	if factoryType.IsVariadic() {
		if factoryType.NumIn() > len(parameters) {
			return newInvalidType(fmt.Errorf("invalid number of input parameters for variadic function: got %d but expected at least %d", len(parameters), factoryType.NumIn()))
		}
	} else {
		if factoryType.NumIn() != len(parameters) {
			return newInvalidType(fmt.Errorf("invalid number of input parameters: got %d but expected %d", len(parameters), factoryType.NumIn()))
		}
	}

	// Use cached reflection operations
	cache := GetGlobalReflectionCache()
	t := &typeFactory{
		factory:     cache.GetValue(function),
		factoryType: factoryType,
	}

	var err error
	t.factoryArguments, err = buildFactoryCallArguments(factoryType, parameters)
	if err != nil {
		return newInvalidType(err)
	}

	return t
}

func buildFactoryCallArguments(t reflect.Type, allParameters []interface{}) ([]reflect.Value, error) {
	actualNumberOfArgs := t.NumIn()

	// Pre-allocate with known size for better performance
	args := make([]reflect.Value, len(allParameters))

	for i, argument := range allParameters {
		var expectedArgumentType reflect.Type
		if t.IsVariadic() && i >= actualNumberOfArgs-1 {
			// variadic argument
			expectedArgumentType = t.In(actualNumberOfArgs - 1).Elem()
		} else {
			// regular argument
			expectedArgumentType = t.In(i)
		}

		// Use cached reflection operations
		cache := GetGlobalReflectionCache()
		args[i] = cache.GetValue(argument)
		if args[i].Kind() != expectedArgumentType.Kind() {
			if stringArg, isString := argument.(string); isString && !IsParameterOrTypeReference(stringArg) {
				return nil, fmt.Errorf("input argument %d is of type %s but needs to be a %s", i+1, args[i].Kind(), expectedArgumentType.Kind())
			}
		}
	}

	return args, nil
}

// Arguments returns all factory parameters from NewType
func (t *typeFactory) Arguments() []interface{} {
	args := make([]interface{}, len(t.factoryArguments))
	for i, argument := range t.factoryArguments {
		args[i] = argument.Interface()
	}
	return args
}

// Generate will instantiate a new instance of the according type.
func (t *typeFactory) Generate(resolver *ParameterResolver) (interface{}, error) {
	args, err := t.generateFactoryArguments(resolver)
	if err != nil {
		return nil, err
	}

	var result []reflect.Value
	if t.factoryType.IsVariadic() {
		result = t.factory.CallSlice(args)
	} else {
		result = t.factory.Call(args)
	}

	// we check the number of return arguments in NewType so there is always exactly one result
	return result[0].Interface(), nil
}

func (t *typeFactory) generateFactoryArguments(resolver *ParameterResolver) ([]reflect.Value, error) {
	if t.factoryType.IsVariadic() {
		return t.generateVariadicFactoryArguments(resolver)
	}

	// Pre-allocate with known size for better performance
	args := make([]reflect.Value, len(t.factoryArguments))
	var err error

	for i, argument := range t.factoryArguments {
		args[i], err = resolver.Resolve(argument, t.factoryType.In(i))

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

func (t *typeFactory) generateVariadicFactoryArguments(resolver *ParameterResolver) ([]reflect.Value, error) {
	numIn := t.factoryType.NumIn()
	// Pre-allocate with known size for better performance
	args := make([]reflect.Value, numIn)
	var err error

	actualNumberOfArgs := t.factoryType.NumIn()
	for i, argument := range t.factoryArguments[:actualNumberOfArgs-1] {
		args[i], err = resolver.Resolve(argument, t.factoryType.In(i))

		switch errorType := err.(type) {
		case nil:
			continue
		case TypeReferenceError:
			return nil, t.invalidReferencedTypeErr(errorType.TypeID, errorType.TypeInstance, i)
		default:
			return nil, err
		}
	}

	n := len(t.factoryArguments) - actualNumberOfArgs + 1
	variadicType := t.factoryType.In(actualNumberOfArgs - 1)
	variadicSlice := reflect.MakeSlice(variadicType, n, n)
	expectedType := variadicType.Elem()
	for i, argument := range t.factoryArguments[actualNumberOfArgs-1:] {
		resolvedArgument, err := resolver.Resolve(argument, expectedType)
		if err != nil {
			switch errorType := err.(type) {
			case TypeReferenceError:
				return nil, t.invalidReferencedTypeErr(errorType.TypeID, errorType.TypeInstance, i)
			default:
				return nil, err
			}
		}

		variadicSlice.Index(i).Set(resolvedArgument)
	}

	args[actualNumberOfArgs-1] = variadicSlice
	return args, nil
}

func (t *typeFactory) invalidReferencedTypeErr(typeID string, typeInstance interface{}, i int) error {
	factoryName := runtime.FuncForPC(t.factory.Pointer()).Name()
	factoryNameParts := strings.Split(factoryName, "/")
	factoryName = factoryNameParts[len(factoryNameParts)-1]

	n := t.factoryType.NumIn()
	factoryArguments := make([]string, n)
	for i := 0; i < n; i++ {
		arg := t.factoryType.In(i)
		factoryArguments[i] = arg.String()
	}

	err := fmt.Errorf("the referenced type \"@%s\" (type %T) can not be passed as argument %d to the function signature %s(%s)",
		typeID, typeInstance, i+1, factoryName, strings.Join(factoryArguments, ", "),
	)

	return err
}
