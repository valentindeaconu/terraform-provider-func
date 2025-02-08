package tftypes

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// IsBoolType checks if a type is a bool.
func IsBoolType(ty attr.Type) bool {
	_, okV := ty.(basetypes.BoolType)
	_, okP := ty.(*basetypes.BoolType)
	return okV || okP
}

// IsNumberType checks if a type is a number.
func IsNumberType(ty attr.Type) bool {
	_, okV := ty.(basetypes.NumberType)
	_, okP := ty.(*basetypes.NumberType)
	return okV || okP
}

// IsStringType checks if a type is a string.
func IsStringType(ty attr.Type) bool {
	_, okV := ty.(basetypes.StringType)
	_, okP := ty.(*basetypes.StringType)
	return okV || okP
}

// IsTupleType checks if a type is a tuple.
func IsTupleType(ty attr.Type) bool {
	_, okV := ty.(basetypes.TupleType)
	_, okP := ty.(*basetypes.TupleType)
	return okV || okP
}

// IsListType checks if a type is a list.
func IsListType(ty attr.Type) bool {
	_, okV := ty.(basetypes.ListType)
	_, okP := ty.(*basetypes.ListType)
	return okV || okP
}

// IsSetType checks if a type is a set.
func IsSetType(ty attr.Type) bool {
	_, okV := ty.(basetypes.SetType)
	_, okP := ty.(*basetypes.SetType)
	return okV || okP
}

// IsObjectType checks if a type is an object.
func IsObjectType(ty attr.Type) bool {
	_, okV := ty.(basetypes.ObjectType)
	_, okP := ty.(*basetypes.ObjectType)
	return okV || okP
}

// IsMapType checks if a type is a map.
func IsMapType(ty attr.Type) bool {
	_, okV := ty.(basetypes.MapType)
	_, okP := ty.(*basetypes.MapType)
	return okV || okP
}

// PlainTypeString takes a type and returns a representative string.
//
// Compared to the built-in String() method of the attr.Type interface,
// this function ignores inner values.
func PlainTypeString(ty attr.Type) string {
	if IsBoolType(ty) {
		return "basetypes.BoolType"
	}

	if IsNumberType(ty) {
		return "basetypes.NumberType"
	}

	if IsStringType(ty) {
		return "basetypes.StringType"
	}

	if IsTupleType(ty) {
		return "basetypes.TupleType"
	}

	if IsListType(ty) {
		return "basetypes.ListType"
	}

	if IsSetType(ty) {
		return "basetypes.SetType"
	}

	if IsObjectType(ty) {
		return "basetypes.ObjectType"
	}

	if IsMapType(ty) {
		return "basetypes.MapType"
	}

	return "basetypes.DynamicType"
}

// TypeEqual checks if two types are equal.
//
// Compared with the built-in Equal method, this method also returns
// true, if one of the types is pointer and the other one not.
func TypeEqual(lhs attr.Type, rhs attr.Type) bool {
	if IsBoolType(lhs) && IsBoolType(rhs) {
		return true
	}

	if IsNumberType(lhs) && IsNumberType(rhs) {
		return true
	}

	if IsStringType(lhs) && IsStringType(rhs) {
		return true
	}

	if IsTupleType(lhs) && IsTupleType(rhs) {
		return true
	}

	if IsListType(lhs) && IsListType(rhs) {
		return true
	}

	if IsSetType(lhs) && IsSetType(rhs) {
		return true
	}

	if IsObjectType(lhs) && IsObjectType(rhs) {
		return true
	}

	if IsMapType(lhs) && IsMapType(rhs) {
		return true
	}

	return false
}
