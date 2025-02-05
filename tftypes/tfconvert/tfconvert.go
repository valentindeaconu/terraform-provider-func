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

// Convert tries to convert a value to a given type
func Convert(ctx context.Context, val attr.Value, typ attr.Type) (attr.Value, error) {
	ty := val.Type(ctx)
	// Same type, return the value
	if ty.Equal(typ) {
		return val, nil
	}

	// Convert into dynamic type
	// Create a dynamic value wrapping the current value
	if (basetypes.DynamicType{}).Equal(typ) {
		return basetypes.NewDynamicValue(val), nil
	}

	// Anything else, convert them if possible
	switch tftypes.PlainTypeString(ty) {
	case "basetypes.BoolType":
		return (&boolConverter{tftypes.EnsurePointer(val).(*basetypes.BoolValue)}).Convert(ctx, typ)
	case "basetypes.NumberType":
		return (&numberConverter{tftypes.EnsurePointer(val).(*basetypes.NumberValue)}).Convert(ctx, typ)
	case "basetypes.StringType":
		return (&stringConverter{tftypes.EnsurePointer(val).(*basetypes.StringValue)}).Convert(ctx, typ)
	case "basetypes.TupleType":
		return (&tupleConverter{tftypes.EnsurePointer(val).(*basetypes.TupleValue)}).Convert(ctx, typ)
	case "basetypes.ListType":
		return (&listConverter{tftypes.EnsurePointer(val).(*basetypes.ListValue)}).Convert(ctx, typ)
	case "basetypes.SetType":
		return (&setConverter{tftypes.EnsurePointer(val).(*basetypes.SetValue)}).Convert(ctx, typ)
	case "basetypes.ObjectType":
		return (&objectConverter{tftypes.EnsurePointer(val).(*basetypes.ObjectValue)}).Convert(ctx, typ)
	case "basetypes.MapType":
		return (&mapConverter{tftypes.EnsurePointer(val).(*basetypes.MapValue)}).Convert(ctx, typ)
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
		return nil, fmt.Errorf("could not convert boolean into %v", typ.String())
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
		return nil, fmt.Errorf("could not convert number into %v", typ.String())
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
		return nil, fmt.Errorf("could not convert string into %v", typ.String())
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
	case "basetypes.SetType":
		cty, err := tftypes.CollapseTypes(v.ElementTypes(ctx))
		if err != nil {
			return nil, fmt.Errorf("cannot convert tuple to set: %w", err)
		}

		return tftypes.DiagnosticsToError(basetypes.NewSetValue(cty, v.Elements()))
	default:
		return nil, fmt.Errorf("could not convert string into %v", typ.String())
	}
}

type listConverter struct {
	*basetypes.ListValue
}

func (v *listConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.TupleType":
		tys := make([]attr.Type, len(v.Elements()))
		for range v.Elements() {
			tys = append(tys, v.ElementType(ctx))
		}

		return tftypes.DiagnosticsToError(basetypes.NewTupleValue(tys, v.Elements()))
	default:
		return nil, fmt.Errorf("could not convert list into %v", typ.String())
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
		for range v.Elements() {
			tys = append(tys, v.ElementType(ctx))
		}

		return tftypes.DiagnosticsToError(basetypes.NewTupleValue(tys, v.Elements()))
	default:
		return nil, fmt.Errorf("could not convert set into %v", typ.String())
	}
}

type objectConverter struct {
	*basetypes.ObjectValue
}

func (v *objectConverter) Convert(ctx context.Context, typ attr.Type) (attr.Value, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.StringType":
		return basetypes.NewStringValue(v.String()), nil
	case "basetypes.MapType":
		cty, err := tftypes.CollapseTypes(maps.Values(v.AttributeTypes(ctx)))
		if err != nil {
			return nil, fmt.Errorf("cannot convert object to map: %w", err)
		}

		return tftypes.DiagnosticsToError(basetypes.NewMapValue(cty, v.Attributes()))
	default:
		return nil, fmt.Errorf("could not object set into %v", typ.String())
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
		for k, _ := range v.Elements() {
			atys[k] = v.ElementType(ctx)
		}

		return tftypes.DiagnosticsToError(basetypes.NewObjectValue(atys, v.Elements()))
	default:
		return nil, fmt.Errorf("could not map set into %v", typ.String())
	}
}
