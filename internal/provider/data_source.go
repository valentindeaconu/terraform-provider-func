package provider

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-func/internal/runtime"
	"terraform-provider-func/tftypes"
	"terraform-provider-func/tftypes/tfconvert"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSource{}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

// DataSource defines the data source implementation.
type DataSource struct {
	funcs map[string]runtime.Function
}

// DataSourceModel describes the data source data model.
type DataSourceModel struct {
	Id     types.String  `tfsdk:"id"`
	Inputs types.Dynamic `tfsdk:"inputs"`
	Result types.Dynamic `tfsdk:"result"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Execute a function via data-source (for pre-v1.8 workflows).",
		MarkdownDescription: "Execute a function via data-source (for pre-v1.8 workflows).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The name of the function.",
				MarkdownDescription: "The name of the function.",
				Required:            true,
			},
			"inputs": schema.DynamicAttribute{
				Description: "Inputs to be fed into the function.",
				MarkdownDescription: strings.Join(
					[]string{
						"Inputs to be fed into the function.",
						"The inputs can either be a tuple (and the order matters! - equivalent of Python's `args`),",
						"or an object with named parameters (the order doesn't matter - equivalent of Python's `kwargs`).",
						"For any other type, the provider will throw an error.",
					},
					" ",
				),
				Required:   true,
				Validators: []validator.Dynamic{&InputsValidator{}},
			},
			"result": schema.DynamicAttribute{
				Description:         "The result of the function. The type will be inferred from the function return type.",
				MarkdownDescription: "The result of the function. The type will be inferred from the function return type.",
				Computed:            true,
			},
		},
	}
}

func (d *DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	funcs, ok := req.ProviderData.(map[string]runtime.Function)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected map[string]runtime.Function, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.funcs = funcs
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If any of the inputs are unknown, we need to defer the execution
	if data.Id.IsUnknown() || data.Inputs.IsUnknown() || data.Inputs.IsUnderlyingValueUnknown() {
		data.Result = basetypes.NewDynamicUnknown()
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	fnName := data.Id.ValueString()

	fn, ok := d.funcs[fnName]
	if !ok {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Function not defined.",
			fmt.Sprintf("There is no function defined with the name '%s'.", fnName),
		)
		return
	}

	params, err := fn.TerraformParameters()
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not find function parameters",
			fmt.Sprintf("Please report this issue to the provider developers. Error: %v", err.Error()),
		)
	}

	ret, err := fn.TerraformReturn()
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not find function return",
			fmt.Sprintf("Please report this issue to the provider developers. Error: %v", err.Error()),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	args := make([]any, len(params))

	val := data.Inputs.UnderlyingValue()
	valTy := val.Type(ctx)
	if tftypes.IsObjectType(valTy) {
		obj := tftypes.EnsurePointer(val).(*basetypes.ObjectValue) //nolint:forcetypeassert

		for k, v := range obj.Attributes() {
			pos := -1
			for i, param := range params {
				if param.GetName() == k {
					pos = i
					break
				}
			}

			if pos == -1 {
				resp.Diagnostics.AddAttributeError(
					path.Root("inputs"),
					"Named parameter not defined.",
					fmt.Sprintf("Function '%s' does not have a parameter called '%s'.", fnName, k),
				)
				return
			}

			if !params[pos].GetType().Equal(v.Type(ctx)) {
				resp.Diagnostics.AddAttributeError(
					path.Root("inputs").AtMapKey(k),
					"Parameter type mismatch.",
					fmt.Sprintf(
						"Parameter '%s' of function '%s' has type '%v', but received a value of type '%v'.",
						k,
						fnName,
						params[pos].GetType().String(),
						v.Type(ctx).String(),
					),
				)
				return
			}

			args[pos] = v
		}
	} else if tftypes.IsTupleType(valTy) {
		tuple := tftypes.EnsurePointer(val).(*basetypes.TupleValue) //nolint:forcetypeassert

		for i, v := range tuple.Elements() {
			if !params[i].GetType().Equal(v.Type(ctx)) {
				resp.Diagnostics.AddAttributeError(
					path.Root("inputs").AtTupleIndex(i),
					"Parameter type mismatch.",
					fmt.Sprintf(
						"Parameter #%d of function '%s' has type '%v', but received a value of type '%v'.",
						i,
						fnName,
						params[i].GetType().String(),
						v.Type(ctx).String(),
					),
				)
				return
			}

			args[i] = v
		}

	} else {
		// We already used a validator to make sure this cannot happen, but just in case, let's throw an error
		resp.Diagnostics.AddAttributeError(
			path.Root("inputs"),
			"Invalid input type.",
			fmt.Sprintf("Inputs of type '%v' are not supported. Expecting objects or tuples.", valTy.String()),
		)
		return
	}

	for i := range args {
		if args[i] == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("inputs"),
				"Missing value for parameter.",
				fmt.Sprintf(
					"Function '%s' have a parameter called '%s' (position %d) but there was no value for it.",
					fnName,
					params[i].GetName(),
					i,
				),
			)
			return
		}
	}

	tflog.Trace(ctx, "calling function", map[string]any{
		"name":       fnName,
		"parameters": args,
	})

	res, err := fn.Execute(args...)
	if err != nil {
		resp.Diagnostics.AddError("Function returned with error.", err.Error())
		return
	}

	resVal := res.(attr.Value) //nolint:forcetypeassert
	if !ret.GetType().Equal(resVal.Type(ctx)) {
		convertedVal, err := tfconvert.Convert(ctx, resVal, ret.GetType())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("result"),
				"Return type mismatch.",
				fmt.Sprintf(
					"Return of function '%s' has type '%v', but received a value of type '%v' (and the provider failed to convert it).",
					fnName,
					ret.GetType().String(),
					resVal.Type(ctx).String(),
				),
			)
			return
		}

		resVal = convertedVal
	}

	data.Result = basetypes.NewDynamicValue(resVal)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type InputsValidator struct{}

func (v *InputsValidator) Description(_ context.Context) string {
	return "inputs must be of type tuple (unnamed) or object (named)"
}

func (v *InputsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *InputsValidator) ValidateDynamic(ctx context.Context, req validator.DynamicRequest, resp *validator.DynamicResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnderlyingValueNull() {
		// Allow null values, this can happen for functions with no inputs
		return
	}

	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsUnderlyingValueUnknown() {
		// Allow unknown values, but keep in mind that we need to defer the function execution too
		return
	}

	value := req.ConfigValue.UnderlyingValue()
	ty := value.Type(ctx)

	switch tftypes.PlainTypeString(ty) {
	case "basetypes.ObjectType":
		// We can accept objects, the user may have passed-in named parameters
		return
	case "basetypes.TupleType":
		// We can accept tuples (although, Terraform core might not be able to send them yet)
		// See https://github.com/hashicorp/terraform-plugin-framework/pull/870
		return
	default:
		break
	}

	resp.Diagnostics.AddError(
		"Invalid input type.",
		fmt.Sprintf("Type %v is not supported for passing in the function inputs", ty.String()),
	)
}
