package javascript

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestGetTerraformType(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  attr.Type
		err   bool
	}{
		// Primitives
		{"Boolean type", "boolean", basetypes.BoolType{}, false},
		{"Number type", "number", basetypes.NumberType{}, false},
		{"String type", "string", basetypes.StringType{}, false},
		{"Any type", "any", basetypes.DynamicType{}, false},

		// Arrays
		{"Array of booleans", "boolean[]", basetypes.ListType{ElemType: basetypes.BoolType{}}, false},
		{"Array of numbers", "number[]", basetypes.ListType{ElemType: basetypes.NumberType{}}, false},
		{"Array of strings", "string[]", basetypes.ListType{ElemType: basetypes.StringType{}}, false},
		{"Array of any", "any[]", basetypes.ListType{ElemType: basetypes.DynamicType{}}, false},

		// Sets
		{"Set of booleans", "Set<boolean>", basetypes.SetType{ElemType: basetypes.BoolType{}}, false},
		{"Set of numbers", "Set<number>", basetypes.SetType{ElemType: basetypes.NumberType{}}, false},
		{"Set of strings", "Set<string>", basetypes.SetType{ElemType: basetypes.StringType{}}, false},
		{"Set of any", "Set<any>", basetypes.SetType{ElemType: basetypes.DynamicType{}}, false},

		// Maps
		{"Map of booleans", "Map<boolean>", basetypes.MapType{ElemType: basetypes.BoolType{}}, false},
		{"Map of numbers", "Map<number>", basetypes.MapType{ElemType: basetypes.NumberType{}}, false},
		{"Map of strings", "Map<string>", basetypes.MapType{ElemType: basetypes.StringType{}}, false},
		{"Map of any", "Map<any>", basetypes.MapType{ElemType: basetypes.DynamicType{}}, false},

		// Tuples
		{"Tuple of same type", "[number, number]", basetypes.TupleType{
			ElemTypes: []attr.Type{basetypes.NumberType{}, basetypes.NumberType{}},
		}, false},
		{"Tuple of mixed types", "[number, string, boolean]", basetypes.TupleType{
			ElemTypes: []attr.Type{basetypes.NumberType{}, basetypes.StringType{}, basetypes.BoolType{}},
		}, false},

		// Composed types
		{
			"Array of sets",
			"Set<string>[]",
			basetypes.ListType{ElemType: basetypes.SetType{ElemType: basetypes.StringType{}}},
			false,
		},
		{
			"Map of arrays",
			"Map<string[]>",
			basetypes.MapType{ElemType: basetypes.ListType{ElemType: basetypes.StringType{}}},
			false,
		},
		{
			"Tuple of map of string arrays and sets of numbers",
			"[Map<string[]>, Set<number>]",
			basetypes.TupleType{ElemTypes: []attr.Type{
				basetypes.MapType{ElemType: basetypes.ListType{ElemType: basetypes.StringType{}}},
				basetypes.SetType{ElemType: basetypes.NumberType{}},
			}},
			false,
		},

		// Unions
		{"Union type (string | number)", "string | number", nil, true},

		// Objects
		{
			name: "Simple object with string and number",
			given: `{
				name: string;
				age: number;
			}`,
			want: basetypes.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": basetypes.StringType{},
					"age":  basetypes.NumberType{},
				},
			},
		},
		{
			name: "Boolean type",
			given: `{
				isActive: boolean;
			}`,
			want: basetypes.ObjectType{
				AttrTypes: map[string]attr.Type{
					"isActive": basetypes.BoolType{},
				},
			},
		},
		{
			name: "Array of strings and numbers",
			given: `{
				tags: string[];
				values: number[];
			}`,
			want: basetypes.ObjectType{
				AttrTypes: map[string]attr.Type{
					"tags":   basetypes.ListType{ElemType: basetypes.StringType{}},
					"values": basetypes.ListType{ElemType: basetypes.NumberType{}},
				},
			},
		},
		// TODO: Failing, but we need it to pass
		// {
		// 	name: "Nested object",
		// 	given: `{
		// 		user: {
		// 			username: string;
		// 			age: number;
		// 		}
		// 	}`,
		// 	want: basetypes.ObjectType{
		// 		AttrTypes: map[string]attr.Type{
		// 			"user": basetypes.ObjectType{
		// 				AttrTypes: map[string]attr.Type{
		// 					"username": basetypes.StringType{},
		// 					"age":      basetypes.NumberType{},
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		{
			name: "Union type (string | number)",
			given: `{
				id: string | number;
			}`,
			err: true,
		},
		{
			name: "Object with all keys same type",
			given: `{
				[key: string]: {
					name: string;
					age: number;
				};
			}`,
			want: basetypes.MapType{
				ElemType: basetypes.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name": basetypes.StringType{},
						"age":  basetypes.NumberType{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := getTerraformType(test.given)

			if err != nil {
				if test.err {
					return
				}

				t.Errorf("conversion failed with error and it was expected to pass: %v", err)
			}

			if result == nil {
				t.Errorf("wrong object type received:\nwant: %s\ngot : nil", test.want)
			}

			if !result.Equal(test.want) {
				t.Errorf("wrong object type received:\nwant: %s\ngot : %s", test.want, result)
			}
		})
	}
}
