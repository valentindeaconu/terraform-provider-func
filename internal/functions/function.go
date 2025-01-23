// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/ssoroka/slice"
)

var (
	_ function.Function = JSFunction{}
)

type JSFunction struct {
	name        string
	callable    Callable
	args        []JSArgument
	returnType  string
	summary     string
	description string
}

func NewFunction() function.Function {
	return JSFunction{}
}

func (r JSFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = r.name
}

func (r JSFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             r.summary,
		MarkdownDescription: r.description,
		Parameters: slice.Map[JSArgument, function.Parameter](r.args, func(arg JSArgument) function.Parameter {
			return arg.ToSDKParameter()
		}),
		Return: (JSArgument{typ: r.returnType}).ToSDKReturn(),
	}
}

func (r JSFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var data []any = make([]any, len(r.args))
	var err error
	for i, arg := range r.args {
		data[i], err = arg.Allocate()
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
	}

	resp.Error = function.ConcatFuncErrors(req.Arguments.Get(ctx, data...))
	if resp.Error != nil {
		return
	}

	res, err := r.callable(data...)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, nil))
	}

	resp.Error = function.ConcatFuncErrors(resp.Result.Set(ctx, res))
}

// func (r JSFunction) createParameterPointers() []any {
// 	return slice.Map[JSArgument, any](r.args, func(arg JSArgument) any {
// 		switch arg.typ {
// 		case "boolean":
// 			var v bool
// 			return &v
// 		case "float64":
// 			var v float64
// 			return &v
// 		case "int64":
// 			var v int64
// 			return &v
// 		case "list":
// 			var v []string = []string{}
// 			return v
// 		case "map":
// 			var v map[string]any = map[string]any{}
// 			return v
// 		case "number":
// 			var v float64
// 			return &v
// 		case "object":
// 			var v map[string]any = map[string]any{}
// 			return v
// 		case "set":
// 			var v map[string]any = map[string]any{}
// 			return v
// 		case "string":
// 			var v string
// 			return &v
// 		case "any":
// 		default:
// 			// Ignore, default return will handle
// 			break
// 		}

// 		var v map[string]any = map[string]any{}
// 		return v
// 	})
// }
