package handler

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// TestNewNullable verifies that NewNullable creates a Nullable with a value
func TestNewNullable(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		n := NewNullable("test")
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != "test" {
			t.Errorf("Expected value 'test', got '%s'", value)
		}
	})

	t.Run("int value", func(t *testing.T) {
		n := NewNullable(42)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != 42 {
			t.Errorf("Expected value 42, got %d", value)
		}
	})

	t.Run("struct value", func(t *testing.T) {
		type TestStruct struct {
			Name string
			Age  int
		}
		s := TestStruct{Name: "Alice", Age: 30}
		n := NewNullable(s)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true")
		}
		result, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result.Name != "Alice" || result.Age != 30 {
			t.Errorf("Expected struct with Name='Alice' Age=30, got %+v", result)
		}
	})

	t.Run("pointer value", func(t *testing.T) {
		str := "hello"
		n := NewNullable(&str)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true")
		}
		result, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil || *result != "hello" {
			t.Error("Expected pointer to 'hello'")
		}
	})

	t.Run("uuid value", func(t *testing.T) {
		id := uuid.New()
		n := NewNullable(id)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != id {
			t.Errorf("Expected UUID %s, got %s", id, value)
		}
	})
}

// TestNil verifies that Nil creates an empty Nullable
func TestNil(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		n := Nil[string]()
		if n.HasValue() {
			t.Error("Expected HasValue() to be false")
		}
	})

	t.Run("int type", func(t *testing.T) {
		n := Nil[int]()
		if n.HasValue() {
			t.Error("Expected HasValue() to be false")
		}
	})

	t.Run("struct type", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}
		n := Nil[TestStruct]()
		if n.HasValue() {
			t.Error("Expected HasValue() to be false")
		}
	})
}

// TestHasValue verifies HasValue returns correct boolean
func TestHasValue(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		n := NewNullable(123)
		if !n.HasValue() {
			t.Error("Expected HasValue() to return true for Nullable with value")
		}
	})

	t.Run("without value", func(t *testing.T) {
		n := Nil[int]()
		if n.HasValue() {
			t.Error("Expected HasValue() to return false for empty Nullable")
		}
	})
}

// TestValue verifies Value returns the contained value or error
func TestValue(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		n := NewNullable("success")
		result, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got '%s'", result)
		}
	})

	t.Run("returns error when value not present", func(t *testing.T) {
		n := Nil[string]()

		value, err := n.Value()
		if err == nil {
			t.Error("Expected error when accessing empty Nullable, got nil")
		}
		if value != "" {
			t.Errorf("Expected zero value (empty string), got '%s'", value)
		}
	})
}

// TestTryValue verifies TryValue returns value and existence boolean
func TestTryValue(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		n := NewNullable(42)
		value, ok := n.TryValue()
		if !ok {
			t.Error("Expected ok to be true")
		}
		if value != 42 {
			t.Errorf("Expected value 42, got %d", value)
		}
	})

	t.Run("without value", func(t *testing.T) {
		n := Nil[int]()
		value, ok := n.TryValue()
		if ok {
			t.Error("Expected ok to be false")
		}
		if value != 0 {
			t.Errorf("Expected zero value 0, got %d", value)
		}
	})

	t.Run("never panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("TryValue should never panic")
			}
		}()

		n := Nil[string]()
		_, _ = n.TryValue()
	})
}

// TestValueOrDefault verifies ValueOrDefault returns value or zero value
func TestValueOrDefault(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		n := NewNullable(100)
		result := n.ValueOrDefault()
		if result != 100 {
			t.Errorf("Expected 100, got %d", result)
		}
	})

	t.Run("returns zero value for int when absent", func(t *testing.T) {
		n := Nil[int]()
		result := n.ValueOrDefault()
		if result != 0 {
			t.Errorf("Expected zero value 0, got %d", result)
		}
	})

	t.Run("returns zero value for string when absent", func(t *testing.T) {
		n := Nil[string]()
		result := n.ValueOrDefault()
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("returns zero value for bool when absent", func(t *testing.T) {
		n := Nil[bool]()
		result := n.ValueOrDefault()
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})

	t.Run("returns zero value for struct when absent", func(t *testing.T) {
		type TestStruct struct {
			Name string
			Age  int
		}
		n := Nil[TestStruct]()
		result := n.ValueOrDefault()
		if result.Name != "" || result.Age != 0 {
			t.Errorf("Expected zero struct, got %+v", result)
		}
	})

	t.Run("returns nil for pointer when absent", func(t *testing.T) {
		n := Nil[*string]()
		result := n.ValueOrDefault()
		if result != nil {
			t.Error("Expected nil pointer")
		}
	})

	t.Run("never panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("ValueOrDefault should never panic")
			}
		}()

		n := Nil[int]()
		_ = n.ValueOrDefault()
	})
}

// TestValueOr verifies ValueOr returns value or provided default
func TestValueOr(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		n := NewNullable(50)
		result := n.ValueOr(99)
		if result != 50 {
			t.Errorf("Expected 50, got %d", result)
		}
	})

	t.Run("returns provided default when absent", func(t *testing.T) {
		n := Nil[int]()
		result := n.ValueOr(99)
		if result != 99 {
			t.Errorf("Expected 99, got %d", result)
		}
	})

	t.Run("string default", func(t *testing.T) {
		n := Nil[string]()
		result := n.ValueOr("default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})

	t.Run("struct default", func(t *testing.T) {
		type TestStruct struct {
			Name string
		}
		n := Nil[TestStruct]()
		defaultStruct := TestStruct{Name: "default"}
		result := n.ValueOr(defaultStruct)
		if result.Name != "default" {
			t.Errorf("Expected default struct, got %+v", result)
		}
	})

	t.Run("never panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("ValueOr should never panic")
			}
		}()

		n := Nil[string]()
		_ = n.ValueOr("fallback")
	})
}

// TestZeroValueBehavior verifies that zero values can be stored
func TestZeroValueBehavior(t *testing.T) {
	t.Run("can store zero int", func(t *testing.T) {
		n := NewNullable(0)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true for zero value")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != 0 {
			t.Error("Expected to retrieve zero value")
		}
	})

	t.Run("can store empty string", func(t *testing.T) {
		n := NewNullable("")
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true for empty string")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != "" {
			t.Error("Expected to retrieve empty string")
		}
	})

	t.Run("can store false", func(t *testing.T) {
		n := NewNullable(false)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true for false")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != false {
			t.Error("Expected to retrieve false")
		}
	})

	t.Run("nil pointer differs from Nil nullable", func(t *testing.T) {
		// This stores "a nil pointer" as a valid value
		var nilPtr *string = nil
		n := NewNullable(nilPtr)
		if !n.HasValue() {
			t.Error("Expected HasValue() to be true even for nil pointer")
		}
		value, err := n.Value()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if value != nil {
			t.Error("Expected to retrieve nil pointer")
		}
	})
}

// TestNullableComparison verifies behavior of different Nullable instances
func TestNullableComparison(t *testing.T) {
	t.Run("two Nullables with same value", func(t *testing.T) {
		n1 := NewNullable(42)
		n2 := NewNullable(42)

		v1, err1 := n1.Value()
		v2, err2 := n2.Value()
		if err1 != nil || err2 != nil {
			t.Errorf("Expected no errors, got err1=%v err2=%v", err1, err2)
		}
		if v1 != v2 {
			t.Error("Expected values to be equal")
		}
	})

	t.Run("two Nil Nullables", func(t *testing.T) {
		n1 := Nil[int]()
		n2 := Nil[int]()

		if n1.HasValue() || n2.HasValue() {
			t.Error("Expected both to have no value")
		}
	})
}

// Benchmark tests
func BenchmarkNewNullable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewNullable(42)
	}
}

func BenchmarkNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Nil[int]()
	}
}

func BenchmarkHasValue(b *testing.B) {
	n := NewNullable(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = n.HasValue()
	}
}

func BenchmarkValue(b *testing.B) {
	n := NewNullable(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = n.Value()
	}
}

func BenchmarkTryValue(b *testing.B) {
	n := NewNullable(42)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = n.TryValue()
	}
}

func BenchmarkValueOrDefault(b *testing.B) {
	n := Nil[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = n.ValueOrDefault()
	}
}

func BenchmarkValueOr(b *testing.B) {
	n := Nil[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = n.ValueOr(99)
	}
}

// Example tests (show up in godoc)

// ExampleNewNullable demonstrates creating a Nullable with a value
func ExampleNewNullable() {
	n := NewNullable("hello")
	if n.HasValue() {
		value, err := n.Value()
		if err == nil {
			fmt.Println(value)
		}
	}
	// Output: hello
}

// ExampleNil demonstrates creating an empty Nullable
func ExampleNil() {
	n := Nil[string]()
	if !n.HasValue() {
		fmt.Println("No value present")
	}
	// Output: No value present
}

// ExampleNullable_Value demonstrates error handling when accessing values
func ExampleNullable_Value() {
	// Safe usage - value is present
	n := NewNullable(42)
	value, err := n.Value()
	if err == nil {
		fmt.Println(value)
	}

	// Handling empty Nullable - returns error:
	empty := Nil[int]()
	_, err = empty.Value()
	if err != nil {
		fmt.Println("Error: value not present")
	}

	// Output: 42
	// Error: value not present
}

// ExampleNullable_TryValue demonstrates safe value access
func ExampleNullable_TryValue() {
	n := NewNullable("success")

	if value, ok := n.TryValue(); ok {
		fmt.Println("Value:", value)
	} else {
		fmt.Println("No value")
	}
	// Output: Value: success
}

// ExampleNullable_ValueOrDefault demonstrates using zero value as fallback
func ExampleNullable_ValueOrDefault() {
	empty := Nil[int]()
	value := empty.ValueOrDefault()
	fmt.Println(value)
	// Output: 0
}

// ExampleNullable_ValueOr demonstrates using custom default
func ExampleNullable_ValueOr() {
	empty := Nil[int]()
	value := empty.ValueOr(10)
	fmt.Println(value)
	// Output: 10
}
