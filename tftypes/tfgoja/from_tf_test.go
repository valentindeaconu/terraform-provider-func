package tfgoja

import (
	"context"
	"math/big"
	"testing"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestFromTfValue(t *testing.T) {
	tests := []struct {
		given attr.Value
		test  string
	}{
		{
			basetypes.NewDynamicNull(),
			`if (v !== null) throw new Error('want null, but got '+v)`,
		},
		{
			basetypes.NewStringValue("hello"),
			`if (v !== 'hello') throw new Error('want "hello", but got '+v)`,
		},
		{
			basetypes.NewBoolValue(true),
			`if (!v) throw new Error('want true, but got '+v)`,
		},
		{
			basetypes.NewBoolValue(false),
			`if (v) throw new Error('want false, but got '+v)`,
		},
		{
			basetypes.NewNumberValue(big.NewFloat(0)),
			`if (v !== 0) throw new Error('want 0, but got '+v)`,
		},
		{
			basetypes.NewNumberValue(big.NewFloat(1)),
			`if (v !== 1) throw new Error('want 1, but got '+v)`,
		},
		{
			basetypes.NewNumberValue(big.NewFloat(1.5)),
			`if (v !== 1.5) throw new Error('want 1.5, but got '+v)`,
		},
		{
			basetypes.NewObjectValueMust(
				map[string]attr.Type{
					"name": basetypes.StringType{},
					"foo":  basetypes.BoolType{},
				},
				map[string]attr.Value{
					"name": basetypes.NewStringValue("Ermintrude"),
					"foo":  basetypes.NewBoolValue(true),
				},
			),
			`
			if (v.name !== 'Ermintrude') throw new Error('wrong name');
			if (v.foo !== true) throw new Error('wrong foo');
			var keys = [];
			for (k in v) {
				keys.push(k);
			}
			if (keys.length != 2 || keys[0] != 'name' || keys[1] != 'foo')
				throw new Error('wrong keys')
			`,
		},
		{
			basetypes.NewMapValueMust(
				basetypes.StringType{},
				map[string]attr.Value{
					"name": basetypes.NewStringValue("Ermintrude"),
				},
			),
			`
			if (v.name !== 'Ermintrude') throw new Error('wrong name');
			var keys = [];
			for (k in v) {
				keys.push(k);
			}
			if (keys.length != 1 || keys[0] != 'name')
				throw new Error('wrong keys')
			`,
		},
		{
			// Empty object
			basetypes.NewObjectValueMust(
				map[string]attr.Type{},
				map[string]attr.Value{},
			),
			`
			if (JSON.stringify(v) != '{}') throw new Error('wrong result');
			`,
		},
		{
			// Empty map
			basetypes.NewMapValueMust(
				basetypes.StringType{},
				map[string]attr.Value{},
			),
			`
			if (JSON.stringify(v) != '{}') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewTupleValueMust(
				[]attr.Type{
					basetypes.BoolType{},
					basetypes.BoolType{},
				},
				[]attr.Value{
					basetypes.NewBoolValue(true),
					basetypes.NewBoolValue(false),
				},
			),
			`
			if (JSON.stringify(v) != '[true,false]') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewListValueMust(
				basetypes.BoolType{},
				[]attr.Value{
					basetypes.NewBoolValue(true),
					basetypes.NewBoolValue(false),
				},
			),
			`
			if (JSON.stringify(v) != '[true,false]') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewSetValueMust(
				basetypes.StringType{},
				[]attr.Value{
					basetypes.NewStringValue("b"),
					basetypes.NewStringValue("a"),
				},
			),
			`
			if (JSON.stringify(v) != '["b","a"]') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewTupleValueMust([]attr.Type{}, []attr.Value{}),
			`
			if (JSON.stringify(v) != '[]') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewListValueMust(basetypes.StringType{}, []attr.Value{}),
			`
			if (JSON.stringify(v) != '[]') throw new Error('wrong result');
			`,
		},
		{
			basetypes.NewSetValueMust(basetypes.StringType{}, []attr.Value{}),
			`
			if (JSON.stringify(v) != '[]') throw new Error('wrong result');
			`,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.given.String(), func(t *testing.T) {
			testJS := goja.New()

			got, err := FromTfValue(ctx, test.given, testJS)
			if err != nil {
				t.Errorf("conversion errored: %s", err.Error())
			}

			if err := testJS.Set("v", got); err != nil {
				t.Errorf("could not set value: %s", err.Error())
			}

			if _, err := testJS.RunString(test.test); err != nil {
				gotObj := got.ToObject(testJS)
				repr, jsonErr := gotObj.MarshalJSON()
				if jsonErr != nil {
					repr = []byte(got.String())
				}

				t.Errorf("assertion failed\nGot:   %s\n%s", repr, err.Error())
			}
		})
	}
}
