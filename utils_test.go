package provider_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/abrosimov/go-provider"
)

func TestGetTypeName(t *testing.T) {
	type myLocalObject struct{}
	type myLocalInterface interface{}
	type myLocalGeneric[T any] struct{}

	myLocalObjectType := provider.GetTypeName[myLocalObject]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.myLocalObject", myLocalObjectType)

	myLocalInterfaceType := provider.GetTypeName[myLocalInterface]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.myLocalInterface", myLocalInterfaceType)

	myLocalGenericTypeForInt := provider.GetTypeName[myLocalGeneric[int]]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.myLocalGeneric[int]", myLocalGenericTypeForInt)

	myLocalGenericTypeForFloat := provider.GetTypeName[myLocalGeneric[float32]]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.myLocalGeneric[float32]", myLocalGenericTypeForFloat)

	MyGlobalObjectType := provider.GetTypeName[MyGlobalObject]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.MyGlobalObject", MyGlobalObjectType)

	myGlobalInterfaceType := provider.GetTypeName[MyGlobalInterface]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.MyGlobalInterface", myGlobalInterfaceType)

	myGlobalGenericTypeForInt := provider.GetTypeName[MyGlobalGeneric[int]]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.MyGlobalGeneric[int]", myGlobalGenericTypeForInt)

	myGlobalGenericTypeForFloat := provider.GetTypeName[MyGlobalGeneric[float32]]()
	require.Equal(t, "github.com/abrosimov/go-provider_test.MyGlobalGeneric[float32]", myGlobalGenericTypeForFloat)
}

func TestIsInterface(t *testing.T) {
	require.False(t, provider.IsInterface[MyGlobalObject]())
	require.False(t, provider.IsInterface[int]())
	require.False(t, provider.IsInterface[MyGlobalGeneric[int]]())
	require.True(t, provider.IsInterface[MyGlobalInterface]())
	require.True(t, provider.IsInterface[fmt.Stringer]())
}

type MyGlobalObject struct {
	name string
}

type MyGlobalInterface interface{}

type MyGlobalGeneric[T any] struct{}

func TestGetCalleeFunc(t *testing.T) {
	require.Equal(t, "github.com/abrosimov/go-provider_test.TestGetCalleeFunc:60", provider.GetCurrentFunc())
	func() {
		require.Equal(t, "github.com/abrosimov/go-provider_test.TestGetCalleeFunc.func1:62", provider.GetCurrentFunc())
	}()
	func() {
		require.Equal(t, "github.com/abrosimov/go-provider_test.TestGetCalleeFunc.func2:65", provider.GetCurrentFunc())
	}()
}

func TestGetCallee(t *testing.T) {
	callee := provider.GetCalleeFunc()
	// Because we want to get name of the function that called TestGetCallee,
	// and we don't want to stick to line numbers in "testing" framework
	// we're sticking only to the function name.
	parts := strings.Split(callee, ":")
	require.Equal(t, "testing.tRunner", parts[0])
	func() {
		require.Equal(t, "github.com/abrosimov/go-provider_test.TestGetCallee:78", provider.GetCalleeFunc())
	}()

	func() {
		require.Equal(t, "github.com/abrosimov/go-provider_test.TestGetCallee:82", provider.GetCalleeFunc())
	}()
}

func TestIsChangesProvider(t *testing.T) {
	type myChangesProvider struct {
		*provider.ChangesNotifier
	}

	require.True(t, provider.IsChangesNotifier[myChangesProvider]())
	require.False(t, provider.IsChangesNotifier[MyGlobalGeneric[int]]())
	require.False(t, provider.IsChangesNotifier[MyGlobalObject]())
	require.False(t, provider.IsChangesNotifier[fmt.Stringer]())
}
