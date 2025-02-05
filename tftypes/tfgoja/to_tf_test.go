package tfgoja

import (
	"context"
	"math/big"
	"testing"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestToTfValue(t *testing.T) {
	tests := []struct {
		Src  string
		Want attr.Value
		Err  bool
	}{
		{
			Src:  "null",
			Want: basetypes.NewDynamicNull(),
		},
		{
			Src:  "undefined",
			Want: basetypes.NewDynamicNull(),
		},
		{
			Src:  "12",
			Want: basetypes.NewNumberValue(big.NewFloat(12)),
		},
		{
			Src:  "12.5",
			Want: basetypes.NewNumberValue(big.NewFloat(12.5)),
		},
		{
			Src:  "true",
			Want: basetypes.NewBoolValue(true),
		},
		{
			Src:  "false",
			Want: basetypes.NewBoolValue(false),
		},
		{
			Src:  `""`,
			Want: basetypes.NewStringValue(""),
		},
		{
			Src:  `"hello"`,
			Want: basetypes.NewStringValue("hello"),
		},
		{
			Src:  `({})`,
			Want: basetypes.NewObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}),
		},
		{
			Src: `({a:"b"})`,
			Want: basetypes.NewObjectValueMust(
				map[string]attr.Type{
					"a": basetypes.StringType{},
				},
				map[string]attr.Value{
					"a": basetypes.NewStringValue("b"),
				},
			),
		},
		{
			Src:  `[]`,
			Want: basetypes.NewTupleValueMust([]attr.Type{}, []attr.Value{}),
		},
		{
			Src: `[true]`,
			Want: basetypes.NewTupleValueMust(
				[]attr.Type{
					basetypes.BoolType{},
				},
				[]attr.Value{
					basetypes.NewBoolValue(true),
				},
			),
		},
		{
			Src:  `(function () {})`,
			Want: basetypes.NewDynamicNull(),
		},
		{
			Src: `new Date(0)`,
			// The Date prototype includes a toJSON function which
			// produces a timestamp string.
			Want: basetypes.NewStringValue("1970-01-01T00:00:00.000Z"),
		},
		{
			Src: `JSON`,
			// The JSON object has no enumerable properties, so it appears
			// as an empty object after conversion.
			Want: basetypes.NewObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}),
		},
		{
			Src:  "NaN",
			Want: basetypes.NewDynamicNull(),
		},
		{
			Src: "Infinity",
			// Even though Terraform can represent positive infinity, JSON doesn't
			// and our mapping is via JSON and so the result is null.
			Want: basetypes.NewDynamicNull(),
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		t.Run(test.Src, func(t *testing.T) {
			js := goja.New()
			result, err := js.RunString(test.Src)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			got, gotErr := ToTfValue(ctx, result, js)
			if test.Err {
				if gotErr == nil {
					t.Errorf("wrong result\ngot:  %#v\nwant: (error)", got)
				}
				return
			}

			if gotErr != nil {
				t.Fatalf("unexpected error\ngot:  %s\nwant: %#v", gotErr, test.Want)
			}

			if !test.Want.Equal(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}
