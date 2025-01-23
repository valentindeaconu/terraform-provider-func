// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JSArgument struct {
	name        string
	description string
	typ         string
}

func (r JSArgument) ToSDKParameter() function.Parameter {
	// Base cases
	switch strings.TrimSpace(r.typ) {
	case "number":
		return function.NumberParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,
		}
	case "string":
		return function.StringParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,
		}
	case "bool":
		return function.BoolParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,
		}
	case "any":
	default:
		break
	}

	// Detect arrays
	if strings.HasSuffix(r.typ, "[]") {
		return function.ListParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,

			ElementType: toAttrType(r.typ[:len(r.typ)-2]),
		}
	}

	// Detect maps
	if strings.HasPrefix(r.typ, "Map<") && strings.HasSuffix(r.typ, ">") {
		return function.MapParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,

			ElementType: toAttrType(r.typ[4 : len(r.typ)-1]),
		}
	}

	// Detect sets
	if strings.HasPrefix(r.typ, "Set<") && strings.HasSuffix(r.typ, ">") {
		return function.SetParameter{
			Name:                r.name,
			Description:         r.description,
			MarkdownDescription: r.description,

			ElementType: toAttrType(r.typ[4 : len(r.typ)-1]),
		}
	}

	return function.DynamicParameter{
		Name:                r.name,
		Description:         r.description,
		MarkdownDescription: r.description,
	}
}

func (r JSArgument) ToSDKReturn() function.Return {
	// Base cases
	switch strings.TrimSpace(r.typ) {
	case "number":
		return function.NumberReturn{}
	case "string":
		return function.StringReturn{}
	case "bool":
		return function.BoolReturn{}
	case "any":
	default:
		break
	}

	// Detect arrays
	if strings.HasSuffix(r.typ, "[]") {
		return function.ListReturn{
			ElementType: toAttrType(r.typ[:len(r.typ)-2]),
		}
	}

	// Detect maps
	if strings.HasPrefix(r.typ, "Map<") && strings.HasSuffix(r.typ, ">") {
		return function.MapReturn{
			ElementType: toAttrType(r.typ[4 : len(r.typ)-1]),
		}
	}

	// Detect sets
	if strings.HasPrefix(r.typ, "Set<") && strings.HasSuffix(r.typ, ">") {
		return function.SetReturn{
			ElementType: toAttrType(r.typ[4 : len(r.typ)-1]),
		}
	}

	return function.DynamicReturn{}
}

func (r JSArgument) Allocate() (any, error) {
	return allocateType(r.typ)
}

func toAttrType(typeStr string) attr.Type {
	// Best cases
	switch typeStr {
	case "number":
		return types.NumberType
	case "string":
		return types.StringType
	case "bool":
		return types.BoolType
	case "any":
		return types.DynamicType
	default:
		break
	}

	// Detect arrays
	if strings.HasSuffix(typeStr, "[]") {
		return types.ListType{
			ElemType: toAttrType(typeStr[0 : len(typeStr)-2]),
		}
	}

	// Detect maps
	if strings.HasPrefix(typeStr, "Map<") && strings.HasSuffix(typeStr, ">") {
		return types.MapType{
			ElemType: toAttrType(typeStr[4 : len(typeStr)-1]),
		}
	}

	// Detect sets
	if strings.HasPrefix(typeStr, "Set<") && strings.HasSuffix(typeStr, ">") {
		return types.SetType{
			ElemType: toAttrType(typeStr[4 : len(typeStr)-1]),
		}
	}

	return types.DynamicType
}

func allocateType(typeStr string) (any, error) {
	typeStr = strings.TrimSpace(typeStr)

	// Base cases
	switch typeStr {
	case "number":
		return new(float64), nil
	case "string":
		return new(string), nil
	case "bool":
		return new(bool), nil
	case "any":
		return new(interface{}), nil
	}

	// Detect arrays (e.g., `number[]`)
	if strings.HasSuffix(typeStr, "[]") {
		elementType := typeStr[:len(typeStr)-2]
		elementObj, err := allocateType(elementType)
		if err != nil {
			return nil, err
		}

		v := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(elementObj).Elem()), 0, 0).Interface()
		return &v, nil
	}

	// Detect maps (e.g., `Map<string>`)
	if strings.HasPrefix(typeStr, "Map<") && strings.HasSuffix(typeStr, ">") {
		innerType := typeStr[4 : len(typeStr)-1]
		innerObj, err := allocateType(innerType)
		if err != nil {
			return nil, err
		}
		valueType := reflect.TypeOf(innerObj).Elem()
		v := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), valueType)).Interface()
		return &v, nil
	}

	// Detect sets (e.g., `Set<string>`)
	if strings.HasPrefix(typeStr, "Set<") && strings.HasSuffix(typeStr, ">") {
		innerType := typeStr[4 : len(typeStr)-1]
		innerObj, err := allocateType(innerType)
		if err != nil {
			return nil, err
		}
		keyType := reflect.TypeOf(innerObj).Elem()
		v := reflect.MakeMap(reflect.MapOf(keyType, reflect.TypeOf(struct{}{}))).Interface()
		return &v, nil
	}

	// Handle nested types (e.g., `Map<Map<string>>`)
	if strings.HasPrefix(typeStr, "Map<") || strings.HasPrefix(typeStr, "Set<") {
		regex := regexp.MustCompile(`^(\w+)<(.+)>$`)
		matches := regex.FindStringSubmatch(typeStr)
		if len(matches) == 3 {
			outerType := matches[1]
			innerType := matches[2]

			if outerType == "Map" {
				innerObj, err := allocateType(innerType)
				if err != nil {
					return nil, err
				}
				innerTypeReflect := reflect.TypeOf(innerObj).Elem()
				v := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(""), innerTypeReflect)).Interface()
				return &v, nil
			} else if outerType == "Set" {
				innerObj, err := allocateType(innerType)
				if err != nil {
					return nil, err
				}
				innerTypeReflect := reflect.TypeOf(innerObj).Elem()
				v := reflect.MakeMap(reflect.MapOf(innerTypeReflect, reflect.TypeOf(struct{}{}))).Interface()
				return &v, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported type: %s", typeStr)
}
