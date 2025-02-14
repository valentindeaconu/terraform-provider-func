package runtime

import (
	"context"
	"terraform-provider-func/tftypes/tfconvert"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	tffunc "github.com/hashicorp/terraform-plugin-framework/function"
)

// Callable represent the bound function signature.
type Callable = func(args ...any) (any, error)

// Function is an abstract interface representing a function.
type Function interface {
	// Name returns the function name
	// It should be formatted as snake case to align with Terraform values, but it is not required.
	Name() string

	// Summary returns a short description of the function
	Summary() string

	// Description returns a long description of the function
	Description() string

	// MarkdownDescription returns a long description of the function
	// formatted as markdown
	MarkdownDescription() string

	// AllocateParameters should allocate objects to which the function parameters can be bound
	AllocateParameters() ([]any, error)

	// ToTerraformParameters returns a list of Terraform parameters that Terraform can understand
	TerraformParameters() ([]tffunc.Parameter, error)

	// TerraformParameters returns a the Terraform return type that Terraform can understand
	TerraformReturn() (tffunc.Return, error)

	// Execute launches the function into execution
	Execute(args ...any) (any, error)
}

// TerraformFunction is a wrapper over the Function interface
// that implements the Terraform's Function interface.
type TerraformFunction struct {
	Function Function
}

func (r TerraformFunction) Metadata(_ context.Context, req tffunc.MetadataRequest, resp *tffunc.MetadataResponse) {
	resp.Name = r.Function.Name()
}

func (r TerraformFunction) Definition(_ context.Context, _ tffunc.DefinitionRequest, resp *tffunc.DefinitionResponse) {
	params, err := r.Function.TerraformParameters()
	if err != nil {
		resp.Diagnostics.AddError("Cannot compute function parameters", err.Error())
		return
	}

	ret, err := r.Function.TerraformReturn()
	if err != nil {
		resp.Diagnostics.AddError("Cannot compute function return", err.Error())
		return
	}

	resp.Definition = tffunc.Definition{
		Summary:             r.Function.Summary(),
		MarkdownDescription: r.Function.Description(),
		Parameters:          params,
		Return:              ret,
	}
}

func (r TerraformFunction) Run(ctx context.Context, req tffunc.RunRequest, resp *tffunc.RunResponse) {
	args, err := r.Function.AllocateParameters()
	if err != nil {
		resp.Error = tffunc.ConcatFuncErrors(resp.Error, tffunc.NewFuncError(err.Error()))
		return
	}

	resp.Error = tffunc.ConcatFuncErrors(req.Arguments.Get(ctx, args...))
	if resp.Error != nil {
		return
	}

	res, err := r.Function.Execute(args...)
	if err != nil {
		resp.Error = tffunc.ConcatFuncErrors(resp.Error, tffunc.NewFuncError(err.Error()))
		return
	}

	rty, err := r.Function.TerraformReturn()
	if err != nil {
		resp.Error = tffunc.ConcatFuncErrors(resp.Error, tffunc.NewFuncError(err.Error()))
		return
	}

	val, err := tfconvert.Convert(ctx, res.(attr.Value), rty.GetType()) //nolint:forcetypeassert
	if err != nil {
		resp.Error = tffunc.ConcatFuncErrors(resp.Error, tffunc.NewFuncError(err.Error()))
		return
	}

	resp.Error = tffunc.ConcatFuncErrors(resp.Result.Set(ctx, val))
}
