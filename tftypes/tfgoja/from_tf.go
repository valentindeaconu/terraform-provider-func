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

	switch {
	case v.IsUnknown():
		return nil, ErrUnknownValue
	case v.IsNull():
		return goja.Null(), nil
	case tftypes.IsObjectType(ty) || tftypes.IsMapType(ty):
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

	switch tftypes.PlainTypeString(ty) {
	case "basetypes.DynamicType":
		return fromTfValueReflect(
			ctx,
			tftypes.EnsurePointer(
				tftypes.EnsurePointer(v).(*basetypes.DynamicValue).UnderlyingValue(), //nolint:forcetypeassert
			),
			js,
		)
	case "basetypes.BoolType":
		return tftypes.EnsurePointer(v).(*basetypes.BoolValue).ValueBool(), nil //nolint:forcetypeassert
	case "basetypes.NumberType":
		raw := tftypes.EnsurePointer(v).(*basetypes.NumberValue).ValueBigFloat() //nolint:forcetypeassert
		if rawInt64, acc := raw.Int64(); acc == big.Exact {
			return rawInt64, nil
		}
		rawFloat, _ := raw.Float64()
		return rawFloat, nil
	case "basetypes.StringType":
		return tftypes.EnsurePointer(v).(*basetypes.StringValue).ValueString(), nil //nolint:forcetypeassert
	case "basetypes.TupleType":
		vv := tftypes.EnsurePointer(v).(*basetypes.TupleValue) //nolint:forcetypeassert

		raw := make([]any, 0, len(vv.Elements()))
		for i, el := range vv.Elements() {
			gojaV, err := FromTfValue(ctx, el, js)
			if err != nil {
				return nil, fmt.Errorf("%w: tuple[%d]: %w", ErrConversionFailure, i, err)
			}

			raw = append(raw, gojaV)
		}

		return raw, nil
	case "basetypes.ListType":
		vv := tftypes.EnsurePointer(v).(*basetypes.ListValue) //nolint:forcetypeassert

		raw := make([]any, 0, len(vv.Elements()))
		for i, el := range vv.Elements() {
			gojaV, err := FromTfValue(ctx, el, js)
			if err != nil {
				return nil, fmt.Errorf("%w: list[%d]: %w", ErrConversionFailure, i, err)
			}

			raw = append(raw, gojaV)
		}

		return raw, nil
	case "basetypes.SetType":
		vv := tftypes.EnsurePointer(v).(*basetypes.SetValue) //nolint:forcetypeassert

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

	return nil, fmt.Errorf("%w: %#v", ErrUnknownType, v)
}

func fromTfValueObject(ctx context.Context, v attr.Value, js *goja.Runtime) (*goja.Object, error) {
	ty := v.Type(ctx)

	var attrs map[string]attr.Value
	var typ string
	if tftypes.IsObjectType(ty) {
		attrs = tftypes.EnsurePointer(v).(*basetypes.ObjectValue).Attributes() //nolint:forcetypeassert
		typ = "object"
	} else if tftypes.IsMapType(ty) {
		attrs = tftypes.EnsurePointer(v).(*basetypes.MapValue).Elements() //nolint:forcetypeassert
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

		if err := ret.DefineDataProperty(k, gojaV, goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_TRUE); err != nil {
			return nil, err
		}
	}

	return ret, nil
}
