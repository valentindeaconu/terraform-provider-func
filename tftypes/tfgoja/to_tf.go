package tfgoja

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
)

// ToTfValue attempts to find an attr.Value that is equivalent to the given
// goja.Value, returning an error if no conversion is possible.
//
// Although Terraform (HCL) is a superset of JSON and thus all Terraform values can
// be converted to JavaScript by way of a JSON-like mapping, JavaScript's type system
// includes many types that have no equivalent in Terraform, such as functions.
//
// For predictability and consistency, the conversion from JavaScript to Terraform
// is defined as a conversion from JavaScript to JSON using the same rules
// as JavaScript's JSON.stringify function, followed by interpretation of that
// result in Terraform using the same rules as the terraform/json package follows.
//
// This function therefore fails in the cases where JSON.stringify would fail.
// Because neither Terraform nor JSON have an equivalent of "undefined", in cases
// where JSON.stringify would return undefined ToTfValue returns a Terraform
// null value.
func ToTfValue(ctx context.Context, v goja.Value, js *goja.Runtime) (attr.Value, error) {
	// There are some exceptions for things that can't be turned into a
	// goja.Object, because they don't have associated boxing prototypes.
	if goja.IsNull(v) || goja.IsUndefined(v) {
		return basetypes.NewDynamicNull(), nil
	}

	// For now at least, the implementation is literally to go via JSON
	// encoding, because goja offers a convenient interface to the same
	// behavior as JSON.stringify.
	src, err := v.ToObject(js).MarshalJSON()
	if err != nil {
		return basetypes.NewDynamicNull(), err
	}

	// If the object cannot be serialized, it will be printed as "null"
	// so we need to return null
	if string(src) == "null" {
		return basetypes.NewDynamicNull(), err
	}

	ty, err := JSONImpliedType(src)
	if err != nil {
		// It'd be weird to end up here because that would suggest that
		// goja's MarshalJSON produced an invalid result, but we'll return
		// it out anyway.
		return basetypes.NewDynamicNull(), err
	}

	var value any = v.Export()

	if vt, ok := value.(time.Time); ok {
		value = vt.UTC().Format("2006-01-02T15:04:05.000Z")
	}

	if _, ok := ty.(basetypes.ObjectType); ok {
		builder := dynamicstruct.NewStruct()

		for k, v := range v.Export().(map[string]any) {
			builder.AddField(
				// We need to title the key to comply with GoLang struct exporting
				cases.Title(language.English, cases.Compact).String(k),
				v,
				// We add tags to make sure other systems parse the key as it is
				fmt.Sprintf(`tfsdk:"%s" json:"%s"`, k, k),
			)
		}

		value = builder.Build().New()

		if err := json.Unmarshal(src, &value); err != nil {
			return nil, err
		}
	}

	var res attr.Value
	if diags := tfsdk.ValueFrom(ctx, value, ty, &res); diags.HasError() {
		var err error = fmt.Errorf("could not reflect goja value into tf")
		for _, diag := range diags {
			err = fmt.Errorf("%v: %v", err, diag.Detail())
		}
		return nil, err
	}

	return res, nil
}
