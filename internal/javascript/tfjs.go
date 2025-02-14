package javascript

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	objectTypeRegExp = regexp.MustCompile(`(?:(\w+)|\[(\w+)\s*:\s*(\w+)\])\s*:\s*({[^}]*}|[\w\[\]{}|]+)\s*;`)
)

// getTerraformType converts a JavaScript (TypeScript) type into a Terraform type.
//
// Complex types are not 100% covered.
// It will return an error if a type that doesn't have an equivalent
// in Terraform is parsed.
func getTerraformType(tys string) (attr.Type, error) {
	// Unions
	if strings.Contains(tys, "|") {
		return nil, fmt.Errorf("union types are not supported")
	}

	// Primitives
	switch tys {
	case "boolean":
		return &basetypes.BoolType{}, nil
	case "number":
		return &basetypes.NumberType{}, nil
	case "string":
		return &basetypes.StringType{}, nil
	case "any", "":
		return &basetypes.DynamicType{}, nil
	default:
		break
	}

	// Arrays
	if strings.HasSuffix(tys, "[]") {
		innerTypeStr := tys[0 : len(tys)-2]
		innerType, err := getTerraformType(innerTypeStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse '%s' type: %w", innerTypeStr, err)
		}

		return &basetypes.ListType{
			ElemType: innerType,
		}, nil
	}

	// Tuples
	if strings.HasPrefix(tys, "[") && strings.HasSuffix(tys, "]") {
		innerTypesStrs := strings.Split(tys[1:len(tys)-1], ",")
		innerTypes := make([]attr.Type, 0, len(innerTypesStrs))

		for _, innerTypeStr := range innerTypesStrs {
			innerTypeStr = strings.TrimSpace(innerTypeStr)

			innerType, err := getTerraformType(innerTypeStr)
			if err != nil {
				return nil, fmt.Errorf("could not parse '%s' type: %w", innerTypeStr, err)
			}

			innerTypes = append(innerTypes, innerType)
		}

		return &basetypes.TupleType{
			ElemTypes: innerTypes,
		}, nil
	}

	// Sets
	if strings.HasPrefix(tys, "Set<") && strings.HasSuffix(tys, ">") {
		innerTypeStr := tys[4 : len(tys)-1]
		innerType, err := getTerraformType(innerTypeStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse '%s' type: %w", innerTypeStr, err)
		}

		return &basetypes.SetType{
			ElemType: innerType,
		}, nil
	}

	// Maps
	if strings.HasPrefix(tys, "Map<") && strings.HasSuffix(tys, ">") {
		innerTypeStr := tys[4 : len(tys)-1]
		innerType, err := getTerraformType(innerTypeStr)
		if err != nil {
			return nil, fmt.Errorf("could not parse '%s' type: %w", innerTypeStr, err)
		}

		return &basetypes.MapType{
			ElemType: innerType,
		}, nil
	}

	// Objects
	if strings.HasPrefix(tys, "{") && strings.HasSuffix(tys, "}") {
		atys := make(map[string]attr.Type)

		matches := objectTypeRegExp.FindAllStringSubmatch(tys, -1)

		if len(matches) == 1 {
			// If only one match is found, and that match is an index signature
			// the object might be able to be converted to a map
			// Otherwise, we need to return with error, since this complex type is
			// not supported

			if match := matches[0]; match[2] != "" && match[3] != "" {
				// Groups 1 and 2 of the regex matches for index signatures,
				// if they are not empty it means we are now analyzing an
				// object that looks like (since we only found one match in total):
				// { [name: type]: type }

				keyName := match[2]
				keyTypStr := match[3]
				valueTypStr := match[4]

				if keyTypStr != "string" {
					return nil, fmt.Errorf(
						"index signatures can only be assigned to maps, which can only have keys of type string, key type: %s",
						keyTypStr,
					)
				}

				typ, err := getTerraformType(valueTypStr)
				if err != nil {
					return nil, fmt.Errorf(
						"could not parse index signature '[%s: %s]' type '%s': %w",
						keyName,
						keyTypStr,
						valueTypStr,
						err,
					)
				}

				return &basetypes.MapType{ElemType: typ}, nil
			}
		}

		for _, match := range matches {
			if match[2] != "" && match[3] != "" {
				// As said previously, we only get 3 matches in case of an index signature.
				// Since this is a complex type (it has other properties than the index signature)
				// we need to return an error because Terraform doesn't support such types.

				return nil, fmt.Errorf("type '%s' is not supported", tys)
			}

			key := match[1]
			typStr := match[4]

			typ, err := getTerraformType(typStr)
			if err != nil {
				return nil, fmt.Errorf("could not parse key '%s' type '%s': %w", key, typStr, err)
			}

			atys[key] = typ
		}

		return &basetypes.ObjectType{AttrTypes: atys}, nil
	}

	return &basetypes.DynamicType{}, nil
}
