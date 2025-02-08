package tftypes

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// EnsurePointer makes sure that the underlying implementation
// of an attr.Value is a pointer.
func EnsurePointer(v attr.Value) attr.Value {
	rv := reflect.ValueOf(v)

	if v == nil {
		return v
	}

	if rv.Kind() == reflect.Ptr {
		return v
	}

	ptr := reflect.New(rv.Type())
	ptr.Elem().Set(rv)

	return ptr.Interface().(attr.Value)
}

// CollapseTypes accepts a slice of types and returns a single type
// if all elements of the slice are of the same type.
func CollapseTypes(tys []attr.Type) (attr.Type, error) {
	var cty attr.Type = nil
	for _, ty := range tys {
		if cty == nil {
			cty = ty
			continue
		}

		if !TypeEqual(ty, cty) {
			return nil, fmt.Errorf("all elements must be of the same type (%v =/= %v)", cty, ty)
		}
	}

	return cty, nil
}

// IgnoreDiagnostics takes as input a value and a Diagnostics object
// and ignores the diagnostics, only returning the value.
func IgnoreDiagnostics[T any](v T, _ diag.Diagnostics) T {
	return v
}

// DiagnosticsToError converts a Diagnostics object into an error.
func DiagnosticsToError[T any](v T, ds diag.Diagnostics) (T, error) {
	if ds.HasError() {
		for _, d := range ds {
			if d.Severity() == diag.SeverityError {
				return v, fmt.Errorf("%s: %s (and %d more errors)", d.Severity(), d.Detail(), len(ds))
			}
		}

		return v, fmt.Errorf("no error diagnostic")
	}

	return v, nil
}
