package javascript

import (
	"context"
	"fmt"
	"terraform-provider-func/internal/runtime"
	"terraform-provider-func/tftypes"
	"terraform-provider-func/tftypes/tfarg"
	"terraform-provider-func/tftypes/tfgoja"

	"github.com/dop251/goja"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	tffunc "github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/ssoroka/slice"
)

// Test that the JavaScriptFunction correctly implements the Function interface.
var (
	_ runtime.Function = &JavaScriptFunction{}
)

// JavaScriptArgument holds the metadata regarding a JS argument.
type JavaScriptArgument struct {
	name        string
	description string
	param       tffunc.Parameter
}

// JavaScriptFunction is a concrete implementation of the Function interface
// and represents a Function that can be executed on a JavaScript runtime.
type JavaScriptFunction struct {
	name        string
	callable    runtime.Callable
	args        []JavaScriptArgument
	ret         tffunc.Return
	summary     string
	description string
}

func (f *JavaScriptFunction) Name() string {
	return f.name
}

func (f *JavaScriptFunction) Summary() string {
	return f.summary
}

func (f *JavaScriptFunction) Description() string {
	return f.description
}

func (f *JavaScriptFunction) MarkdownDescription() string {
	return f.description
}

func (f *JavaScriptFunction) AllocateParameters() ([]any, error) {
	var data []any = make([]any, len(f.args))

	for i, arg := range f.args {
		data[i] = tftypes.EnsurePointer(arg.param.GetType().ValueType(context.Background()))
	}

	return data, nil
}

func (f *JavaScriptFunction) TerraformParameters() ([]tffunc.Parameter, error) {
	return slice.Map[JavaScriptArgument, tffunc.Parameter](f.args, func(arg JavaScriptArgument) tffunc.Parameter {
		return arg.param
	}), nil
}

func (f *JavaScriptFunction) TerraformReturn() (tffunc.Return, error) {
	return f.ret, nil
}

func (f *JavaScriptFunction) Execute(args ...any) (any, error) {
	return f.callable(args...)
}

type javaScriptArgumentInput struct {
	name        string
	description string
	jsType      string
}

type javascriptFunctionInput struct {
	name        string
	summary     string
	description string
	args        []javaScriptArgumentInput
	retJsType   string
	callable    goja.Callable
}

// NewJavaScriptFunction creates a new JavaScriptFunction.
func NewJavaScriptFunction(in *javascriptFunctionInput, runtime *goja.Runtime) (*JavaScriptFunction, error) {
	if in == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if in.name == "" {
		return nil, fmt.Errorf("a function without a name cannot exist")
	}

	args := make([]JavaScriptArgument, len(in.args))
	for i, arg := range in.args {
		if arg.name == "" {
			return nil, fmt.Errorf("argument %d of function %s does not have a name", i, in.name)
		}

		taty, err := getTerraformType(arg.jsType)
		if err != nil {
			return nil, fmt.Errorf("argument type %d of function %s is not Terraform-compatible: %w", i, in.name, err)
		}

		p, err := tfarg.AsTerraformParameter(taty, arg.name, &tfarg.ParameterOptions{
			Description:         arg.description,
			MarkdownDescription: arg.description,
		})
		if err != nil {
			return nil, fmt.Errorf("argument %d of function %s cannot be converted to Terraform param: %w", i, in.name, err)
		}

		args[i] = JavaScriptArgument{
			name:        arg.name,
			description: arg.description,
			param:       p,
		}
	}

	trty, err := getTerraformType(in.retJsType)
	if err != nil {
		return nil, fmt.Errorf("return type of function %s is not Terraform-compatible: %w", in.name, err)
	}

	ret, err := tfarg.AsTerraformReturn(trty)
	if err != nil {
		return nil, fmt.Errorf("return of function %s cannot be converted to Terraform: %w", in.name, err)
	}

	return &JavaScriptFunction{
		name:        in.name,
		summary:     in.summary,
		description: in.description,
		args:        args,
		ret:         ret,
		callable:    bindCallableToRuntime(runtime, in.callable),
	}, nil
}

func bindCallableToRuntime(runtime *goja.Runtime, callable goja.Callable) runtime.Callable {
	ctx := context.Background()

	return func(args ...any) (any, error) {
		gojaArgs := make([]goja.Value, len(args))

		for i, arg := range args {
			res, err := tfgoja.FromTfValue(ctx, arg.(attr.Value), runtime) //nolint:forcetypeassert
			if err != nil {
				return nil, fmt.Errorf("argument %d cannot be converted to Terraform: %w", i, err)
			}

			gojaArgs[i] = res
		}

		res, err := callable(goja.Undefined(), gojaArgs...)
		if err != nil {
			return nil, fmt.Errorf("func exec: %w", err)
		}

		tfValue, err := tfgoja.ToTfValue(ctx, res, runtime)
		if err != nil {
			return nil, fmt.Errorf("return cannot be converted to Terraform: %w", err)
		}

		return tfValue, err
	}
}
