package tfconvert

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"terraform-provider-func/tftypes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"golang.org/x/exp/maps"
)

type ConverterFrom interface {
	Convert(context.Context, attr.Type) (attr.Value, error)
}

// Convert tries to convert a value to a given type.
func Convert(ctx context.Context, val attr.Value, typ attr.Type) (attr.Value, error) {
	ty := val.Type(ctx)

	// Same type, return the value
	if tftypes.TypeEqual(ty, typ) {
		return val, nil
	}

	// Convert into dynamic type
	// Create a dynamic value wrapping the current value
	if tftypes.TypeEqual(basetypes.DynamicType{}, typ) {
		return basetypes.NewDynamicValue(val), nil
	}

	// Anything else, convert them if possible
	switch tftypes.PlainTypeString(ty) {
	case "basetypes.BoolType":
		return (&boolConverter{tftypes.EnsurePointer(val).(*basetypes.BoolValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.NumberType":
		return (&numberConverter{tftypes.EnsurePointer(val).(*basetypes.NumberValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.StringType":
		return (&stringConverter{tftypes.EnsurePointer(val).(*basetypes.StringValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.TupleType":
		return (&tupleConverter{tftypes.EnsurePointer(val).(*basetypes.TupleValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.ListType":
		return (&listConverter{tftypes.EnsurePointer(val).(*basetypes.ListValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.SetType":
		return (&setConverter{tftypes.EnsurePointer(val).(*basetypes.SetValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.ObjectType":
		return (&objectConverter{tftypes.EnsurePointer(val).(*basetypes.ObjectValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	case "basetypes.MapType":
		return (&mapConverter{tftypes.EnsurePointer(val).(*basetypes.MapValue)}).Convert(ctx, typ) //nolint:forcetypeassert
	}

	return nil, fmt.Errorf("don't know how to convert %s into %s", ty, typ)
}

type boolConverter struct {
	*basetypes.BoolValue
}

func (v *boolConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.NumberType":
		return basetypes.NewNumberValue(big.NewFloat(map[bool]float64{false: 0.0, true: 1.0}[v.ValueBool()])), nil
	case "basetypes.StringType":
		return basetypes.NewStringValue(map[bool]string{false: "false", true: "true"}[v.ValueBool()]), nil
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type numberConverter struct {
	*basetypes.NumberValue
}

func (v *numberConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.BoolType":
		raw := v.ValueBigFloat()
		if rawInt64, acc := raw.Int64(); acc == big.Exact {
			return basetypes.NewBoolValue(rawInt64 != 0), nil
		}
		rawFloat, acc := raw.Float64()
		return basetypes.NewBoolValue(math.Abs(rawFloat) < math.Pow10(-int(acc))), nil
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.ValueBigFloat().String()), nil
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type stringConverter struct {
	*basetypes.StringValue
}

func (v *stringConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type tupleConverter struct {
	*basetypes.TupleValue
}

func (v *tupleConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.ListType":
		cty, err := tftypes.CollapseTypes(v.ElementTypes(ctx))
		if err != nil {
			return nil, fmt.Errorf("cannot convert tuple to list: %w", err)
		}

		return tftypes.DiagnosticsToError(basetypes.NewListValue(cty, v.Elements()))
	case "basetypes.TupleType":
		target := tftypes.EnsureTypePointer(typ).(*basetypes.TupleType).ElementTypes() //nolint:forcetypeassert
		current := v.ElementTypes(ctx)

		if len(target) != len(current) {
			return nil, fmt.Errorf("cannot convert to tuple type with fewer elements than the existing one")
		}

		hasDiff := false
		for i := range current {
			if !current[i].Equal(target[i]) {
				hasDiff = true
				break
			}
		}

		if !hasDiff {
			return v.TupleValue, nil
		}

		currentElements := v.Elements()
		convertedElements := make([]attr.Value, len(target))
		for i := range current {
			if current[i].Equal(target[i]) {
				convertedElements[i] = currentElements[i]
			} else {
				val, err := Convert(ctx, currentElements[i], target[i])
				if err != nil {
					return nil, fmt.Errorf("cannot convert tuple index '%d': %v", i, err)
				}

				convertedElements[i] = val
			}
		}

		return tftypes.DiagnosticsToError(basetypes.NewTupleValue(target, convertedElements))
	case "basetypes.SetType":
		cty, err := tftypes.CollapseTypes(v.ElementTypes(ctx))
		if err != nil {
			return nil, fmt.Errorf("cannot convert tuple to set: %w", err)
		}

		return tftypes.DiagnosticsToError(basetypes.NewSetValue(cty, v.Elements()))
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type listConverter struct {
	*basetypes.ListValue
}

func (v *listConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.ListType":
		target := tftypes.EnsureTypePointer(typ).(*basetypes.ListType).ElementType() //nolint:forcetypeassert
		if v.ElementType(ctx).Equal(target) {
			return v.ListValue, nil
		}

		convertedElements := make([]attr.Value, len(v.Elements()))
		for i, value := range v.Elements() {
			convertedValue, err := Convert(ctx, value, target)
			if err != nil {
				return nil, fmt.Errorf("cannot convert list index %d: %v", i, err)
			}

			convertedElements[i] = convertedValue
		}

		return tftypes.DiagnosticsToError(basetypes.NewListValue(typ, convertedElements))
	case "basetypes.TupleType":
		tys := make([]attr.Type, len(v.Elements()))
		for i := range v.Elements() {
			tys[i] = v.ElementType(ctx)
		}

		return tftypes.DiagnosticsToError(basetypes.NewTupleValue(tys, v.Elements()))
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type setConverter struct {
	*basetypes.SetValue
}

func (v *setConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.ListType":
		return tftypes.DiagnosticsToError(basetypes.NewListValue(v.ElementType(ctx), v.Elements()))
	case "basetypes.TupleType":
		tys := make([]attr.Type, len(v.Elements()))
		for i := range v.Elements() {
			tys[i] = v.ElementType(ctx)
		}

		return tftypes.DiagnosticsToError(basetypes.NewTupleValue(tys, v.Elements()))
	case "basetypes.SetType":
		target := tftypes.EnsureTypePointer(typ).(*basetypes.SetType).ElementType() //nolint:forcetypeassert
		if v.ElementType(ctx).Equal(target) {
			return v.SetValue, nil
		}

		convertedElements := make([]attr.Value, len(v.Elements()))
		for i, value := range v.Elements() {
			convertedValue, err := Convert(ctx, value, target)
			if err != nil {
				return nil, fmt.Errorf("cannot convert set index %d (value %s): %v", i, value.String(), err)
			}

			convertedElements[i] = convertedValue
		}

		return tftypes.DiagnosticsToError(basetypes.NewSetValue(typ, convertedElements))
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type objectConverter struct {
	*basetypes.ObjectValue
}

func (v *objectConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.ObjectType":
		target := tftypes.EnsureTypePointer(typ).(*basetypes.ObjectType).AttributeTypes() //nolint:forcetypeassert
		current := v.AttributeTypes(ctx)

		if len(target) != len(current) {
			return nil, fmt.Errorf("cannot convert to object type with fewer keys than the existing one")
		}

		for k := range current {
			if _, ok := target[k]; !ok {
				return nil, fmt.Errorf("target type is missing the '%s' key", k)
			}
		}

		// No need to check if a target key is not in current because
		// we already checked the length of the maps and at this point,
		// they are equal, so if the length check passes and all the
		// keys of current are in target, we can assume they both
		// share the same keys.

		hasDiff := false
		for k := range current {
			if !current[k].Equal(target[k]) {
				hasDiff = true
				break
			}
		}

		if !hasDiff {
			return v.ObjectValue, nil
		}

		currentAttributes := v.Attributes()
		convertedAttributes := make(map[string]attr.Value, len(target))
		for k := range current {
			if current[k].Equal(target[k]) {
				convertedAttributes[k] = currentAttributes[k]
			} else {
				val, err := Convert(ctx, currentAttributes[k], target[k])
				if err != nil {
					return nil, fmt.Errorf("cannot convert object key '%s': %v", k, err)
				}

				convertedAttributes[k] = val
			}
		}

		return tftypes.DiagnosticsToError(basetypes.NewObjectValue(target, convertedAttributes))
	case "basetypes.MapType":
		cty, err := tftypes.CollapseTypes(maps.Values(v.AttributeTypes(ctx)))
		if err != nil {
			return nil, fmt.Errorf("cannot convert object to map: %w", err)
		}

		return tftypes.DiagnosticsToError(basetypes.NewMapValue(cty, v.Attributes()))
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}

type mapConverter struct {
	*basetypes.MapValue
}

func (v *mapConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.ObjectType":
		atys := make(map[string]attr.Type, len(v.Elements()))
		for k := range v.Elements() {
			atys[k] = v.ElementType(ctx)
		}

		return tftypes.DiagnosticsToError(basetypes.NewObjectValue(atys, v.Elements()))
	case "basetypes.MapType":
		target := tftypes.EnsureTypePointer(typ).(*basetypes.MapType).ElementType() //nolint:forcetypeassert
		if v.ElementType(ctx).Equal(target) {
			return v.MapValue, nil
		}

		convertedElements := make(map[string]attr.Value, len(v.Elements()))
		for key, value := range v.Elements() {
			convertedValue, err := Convert(ctx, value, target)
			if err != nil {
				return nil, fmt.Errorf("cannot convert map key '%s': %v", key, err)
			}

			convertedElements[key] = convertedValue
		}

		return tftypes.DiagnosticsToError(basetypes.NewMapValue(typ, convertedElements))
	default:
		return nil, fmt.Errorf("could not convert %v into %v", v.Type(ctx).String(), typ.String())
	}
}
