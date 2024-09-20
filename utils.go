package provider

import (
	"fmt"
	"reflect"
	"runtime"
)

func getType[T any]() reflect.Type {
	/*
		A bit of dark magic:
		if T is a regular type we're able to instantiate it as `var val T` and get its fully
		qualified name via reflect.TypeOf().
		However, if T is an interface we can't just instantiate it because:
			1. its value will be `nil` and attempt to call its methods will panic.
			2. information about interface name won't be available through reflection.

		So we're creating the zero-slice which takes 0 memory, but keeps the information about
		underlying type (regular, interface or even generic) and then retrieving this information
		through reflections.
		var zero [0]T
		return reflect.TypeOf(zero).Elem()
	*/
	return reflect.TypeFor[T]()
}

func IsInterface[T any]() bool {
	t := getType[T]()
	return t.Kind() == reflect.Interface
}

func IsChangesNotifier[T any]() bool {
	t := getType[T]()
	if t.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Type == reflect.TypeOf((*ChangesNotifier)(nil)) {
			return true
		}
	}
	return false
}

func GetTypeName[T any]() string {
	t := getType[T]()
	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
}

// GetCurrentFunc return func name that called callee of this func.
func GetCurrentFunc() string {
	const currentFuncPositionInStack = 3
	return getFuncNameInStack(currentFuncPositionInStack)
}

func GetCalleeFunc() string {
	const calleeFuncPositionInStack = 4
	return getFuncNameInStack(calleeFuncPositionInStack)
}

func getFuncNameInStack(skip int) string {
	const stackSize = 15
	pc := make([]uintptr, stackSize)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return fmt.Sprintf("%s:%d", frame.Function, frame.Line)
}

func getStackTrace() string {
	const maxStackSize = 4096
	b := make([]byte, maxStackSize)
	n := runtime.Stack(b, false)
	s := string(b[:n])
	return s
}
