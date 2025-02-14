package tfarg

import (
	"fmt"
	"terraform-provider-func/tftypes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ParameterOptions struct {
	Description         string
	MarkdownDescription string
}

// AsTerraformParameter takes in a Terraform type and generates
// a parameter based on that type.
func AsTerraformParameter(typ attr.Type, name string, in *ParameterOptions) (function.Parameter, error) {
	if in == nil {
		in = &ParameterOptions{}
	}

	switch tftypes.PlainTypeString(typ) {
	case "basetypes.BoolType":
		return &function.BoolParameter{
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.NumberType":
		return &function.NumberParameter{
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.StringType":
		return &function.StringParameter{
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.TupleType":
		return nil, fmt.Errorf("tuples cannot be configured as function parameters")
	case "basetypes.ListType":
		return &function.ListParameter{
			ElementType:         typ.(*basetypes.ListType).ElemType, //nolint:forcetypeassert
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.SetType":
		return &function.SetParameter{
			ElementType:         typ.(*basetypes.SetType).ElemType, //nolint:forcetypeassert
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.ObjectType":
		return &function.ObjectParameter{
			AttributeTypes:      typ.(*basetypes.ObjectType).AttrTypes, //nolint:forcetypeassert
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	case "basetypes.MapType":
		return &function.MapParameter{
			ElementType:         typ.(*basetypes.MapType).ElemType, //nolint:forcetypeassert
			AllowNullValue:      true,
			AllowUnknownValues:  false,
			Name:                name,
			Description:         in.Description,
			MarkdownDescription: in.MarkdownDescription,
		}, nil
	default:
		break
	}

	return &function.DynamicParameter{
		AllowNullValue:      true,
		AllowUnknownValues:  false,
		Name:                name,
		Description:         in.Description,
		MarkdownDescription: in.MarkdownDescription,
	}, nil
}

// AsTerraformReturn takes in a Terraform type and generates
// a return based on that type.
func AsTerraformReturn(typ attr.Type) (function.Return, error) {
	switch tftypes.PlainTypeString(typ) {
	case "basetypes.BoolType":
		return &function.BoolReturn{}, nil
	case "basetypes.NumberType":
		return &function.NumberReturn{}, nil
	case "basetypes.StringType":
		return &function.StringReturn{}, nil
	case "basetypes.TupleType":
		return nil, fmt.Errorf("tuples cannot be configured as function return")
	case "basetypes.ListType":
		return &function.ListReturn{
			ElementType: typ.(*basetypes.ListType).ElemType, //nolint:forcetypeassert
		}, nil
	case "basetypes.SetType":
		return &function.SetReturn{
			ElementType: typ.(*basetypes.SetType).ElemType, //nolint:forcetypeassert
		}, nil
	case "basetypes.ObjectType":
		return &function.ObjectReturn{
			AttributeTypes: typ.(*basetypes.ObjectType).AttrTypes, //nolint:forcetypeassert
		}, nil
	case "basetypes.MapType":
		return &function.MapReturn{
			ElementType: typ.(*basetypes.MapType).ElemType, //nolint:forcetypeassert
		}, nil
	default:
		break
	}

	return &function.DynamicReturn{}, nil
}
