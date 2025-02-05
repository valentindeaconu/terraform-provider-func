//
// This package is a replica of zclconf/go-cty-goja
// Source: https://github.com/zclconf/go-cty-goja/blob/0e16cd613893f32bdad7d23238267bf9a4d2c74f/ctygoja/from_cty.go#L1C1-L140C2
// It is adapted to use newer types from the terraform-plugin-framework
//

package tfgoja

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"terraform-provider-func/tftypes"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	ErrUnknownValue      = errors.New("cannot convert an unknown value")
	ErrUnknownType       = errors.New("don't know how to convert type")
	ErrConversionFailure = errors.New("cannot convert value")
)

// FromTfValue takes an attr.Value and returns the equivalent goja.Value
// belonging to the given goja Runtime.
//
// Only known values can be converted to goja.Value. If you pass an unknown
// value then this function will return an error. This function cannot convert
// capsule-typed values and will return an error if you pass one.
//
// The conversions from attr.Value to JavaScript follow similar rules as the default
// representation of Terraform in JSON and so a round-trip through goja.Value and
// back to attr.Value is lossy: maps will generalize as objects and lists and
// sets will generalize as tuples.
//
// This function must not be called concurrently with other use of the given
// runtime.
func FromTfValue(ctx context.Context, v attr.Value, js *goja.Runtime) (goja.Value, error) {
	ty := v.Type(ctx)

	_, isObjectType := ty.(basetypes.ObjectType)
	_, isMapType := ty.(basetypes.MapType)

	switch {
	case v.IsUnknown():
		return nil, ErrUnknownValue
	case v.IsNull():
		return goja.Null(), nil
	case isObjectType || isMapType:
		return fromTfValueObject(ctx, v, js)
	default:
		raw, err := fromTfValueReflect(ctx, v, js)
		if err != nil {
			return nil, err
		}
		return js.ToValue(raw), nil
	}
}

func fromTfValueReflect(ctx context.Context, v attr.Value, js *goja.Runtime) (any, error) {
	ty := v.Type(ctx)

	if _, ok := ty.(basetypes.DynamicType); ok {
		return fromTfValueReflect(
			ctx,
			tftypes.EnsurePointer(
				tftypes.EnsurePointer(v).(*basetypes.DynamicValue).UnderlyingValue(),
			),
			js,
		)
	}

	if _, ok := ty.(basetypes.BoolType); ok {
		return tftypes.EnsurePointer(v).(*basetypes.BoolValue).ValueBool(), nil
	}

	if _, ok := ty.(basetypes.StringType); ok {
		return tftypes.EnsurePointer(v).(*basetypes.StringValue).ValueString(), nil
	}

	if _, ok := ty.(basetypes.NumberType); ok {
		raw := tftypes.EnsurePointer(v).(*basetypes.NumberValue).ValueBigFloat()
		if rawInt64, acc := raw.Int64(); acc == big.Exact {
			return rawInt64, nil
		}
		rawFloat, _ := raw.Float64()
		return rawFloat, nil
	}

	if _, ok := ty.(basetypes.ListType); ok {
		vv := tftypes.EnsurePointer(v).(*basetypes.ListValue)

		raw := make([]any, 0, len(vv.Elements()))
		for i, el := range vv.Elements() {
			gojaV, err := FromTfValue(ctx, el, js)
			if err != nil {
				return nil, fmt.Errorf("%w: list[%d]: %w", ErrConversionFailure, i, err)
			}

			raw = append(raw, gojaV)
		}

		return raw, nil
	}

	if _, ok := ty.(basetypes.SetType); ok {
		vv := tftypes.EnsurePointer(v).(*basetypes.SetValue)

		raw := make([]any, 0, len(vv.Elements()))
		for i, el := range vv.Elements() {
			gojaV, err := FromTfValue(ctx, el, js)
			if err != nil {
				return nil, fmt.Errorf("%w: set[%d]: %w", ErrConversionFailure, i, err)
			}

			raw = append(raw, gojaV)
		}

		return raw, nil
	}

	if _, ok := ty.(basetypes.TupleType); ok {
		vv := tftypes.EnsurePointer(v).(*basetypes.TupleValue)

		raw := make([]any, 0, len(vv.Elements()))
		for i, el := range vv.Elements() {
			gojaV, err := FromTfValue(ctx, el, js)
			if err != nil {
				return nil, fmt.Errorf("%w: tuple[%d]: %w", ErrConversionFailure, i, err)
			}

			raw = append(raw, gojaV)
		}

		return raw, nil
	}

	return nil, fmt.Errorf("%w: %#v", ErrUnknownType, v)
}

func fromTfValueObject(ctx context.Context, v attr.Value, js *goja.Runtime) (*goja.Object, error) {
	ty := v.Type(ctx)

	var attrs map[string]attr.Value
	var typ string
	if _, ok := ty.(basetypes.ObjectType); ok {
		attrs = tftypes.EnsurePointer(v).(*basetypes.ObjectValue).Attributes()
		typ = "object"
	} else if _, ok := ty.(basetypes.MapType); ok {
		attrs = tftypes.EnsurePointer(v).(*basetypes.MapValue).Elements()
		typ = "map"
	} else {
		return nil, fmt.Errorf("%w: '%v' should be map or object, but it is not", ErrUnknownType, ty)
	}

	ret := js.NewObject()
	for k, v := range attrs {
		gojaV, err := FromTfValue(ctx, v, js)
		if err != nil {
			return nil, fmt.Errorf("%w: %s[%s]: %w", ErrConversionFailure, typ, k, err)
		}

		ret.DefineDataProperty(k, gojaV, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE)
	}

	return ret, nil
}
