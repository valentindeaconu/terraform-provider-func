// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"github.com/ssoroka/slice"
)

//go:embed metadata.js
var metadataScript string

type Callable = func(args ...any) (any, error)

type Parser struct {
	vm *goja.Runtime

	funcs []JSFunction
}

func New() *Parser {
	vm := goja.New()

	return &Parser{
		vm:    vm,
		funcs: []JSFunction{},
	}
}

func (p *Parser) Parse(script string) error {
	_, err := p.vm.RunString(fmt.Sprintf("%s\n\n\n%s", script, metadataScript))
	if err != nil {
		return fmt.Errorf("could not run user-defined script: %v", err)
	}

	exports := p.vm.Get("__exports").ToObject(p.vm)

	for _, key := range exports.Keys() {
		fn := exports.Get(key).ToObject(p.vm)

		name := key

		var funcStringValue goja.Value
		if err := p.vm.Try(func() { funcStringValue = fn.Get("fnString") }); err != nil {
			return fmt.Errorf("function %s cannot be stringified: %w", key, err)
		} else if funcStringValue == nil {
			return fmt.Errorf("function %s has an empty stringified value", key)
		}
		funcString := funcStringValue.String()

		argNames, err := extractArgNames(funcString)
		if err != nil {
			return fmt.Errorf("could no extract argument names from function %s: %w", key, err)
		}

		var rawArgsValue goja.Value
		if err := p.vm.Try(func() { rawArgsValue = fn.Get("args") }); err != nil {
			return fmt.Errorf("function %s is has an invalid 'args' declarations: %w", key, err)
		} else if rawArgsValue == nil {
			return fmt.Errorf("function %s is missing the required 'args' declarations", key)
		}

		exportedRawArgs := rawArgsValue.Export()
		rawArgs, ok := exportedRawArgs.([]any)
		if !ok {
			return fmt.Errorf("function %s is has an invalid 'args' declarations: expecting array of args, but instead got %T", key, exportedRawArgs)
		}

		var args []JSArgument = make([]JSArgument, 0, len(rawArgs))
		for i, rawArg := range rawArgs {
			if arg, ok := rawArg.(map[string]any); ok {
				jsArg := JSArgument{}

				if val, ok := arg["type"].(string); ok {
					jsArg.typ = val
				} else {
					return fmt.Errorf("argument %d of function %s is missing the required 'type' declaration", i, key)
				}

				jsArg.name = argNames[i]

				if val, ok := arg["description"].(string); ok {
					jsArg.description = val
				}

				args = append(args, jsArg)
			} else {
				return fmt.Errorf("could not parse argument %d of function %s", i, key)
			}
		}

		var returnTypeValue goja.Value
		if err := p.vm.Try(func() { returnTypeValue = fn.Get("returns") }); err != nil {
			return fmt.Errorf("function %s is has an invalid 'returns' type declaration: %w", key, err)
		} else if returnTypeValue == nil {
			return fmt.Errorf("function %s is missing the required 'returns' type declaration", key)
		}
		returnType := returnTypeValue.String()

		var funcValue goja.Value
		if err := p.vm.Try(func() { funcValue = fn.Get("fn") }); err != nil {
			return fmt.Errorf("function %s is has an invalid 'fn' function declaration: %w", key, err)
		} else if funcValue == nil {
			return fmt.Errorf("function %s is missing the required 'fn' function declaration", key)
		}

		callable, ok := goja.AssertFunction(funcValue)
		if !ok {
			return fmt.Errorf("library exported %v func, but the JSVM could not return its pointer", name)
		}

		var summaryValue goja.Value
		p.vm.Try(func() { summaryValue = fn.Get("summary") })
		var summary string
		if summaryValue != nil {
			summary = summaryValue.String()
		}

		var descriptionValue goja.Value
		p.vm.Try(func() { descriptionValue = fn.Get("description") })
		var description string
		if descriptionValue != nil {
			description = descriptionValue.String()
		}

		p.funcs = append(p.funcs, JSFunction{
			name:       name,
			args:       args,
			returnType: returnType,
			// Wrap the function to force "this" to undefined and cast args and return
			callable: func(args ...any) (any, error) {
				gojaArgs := slice.Map[any, goja.Value](args, func(arg any) goja.Value {
					return p.vm.ToValue(arg)
				})

				res, err := callable(goja.Undefined(), gojaArgs...)
				return res.Export(), err
			},
			summary:     summary,
			description: description,
		})
	}

	return nil
}

func (p *Parser) GetFunctions() []JSFunction {
	return p.funcs
}

func extractArgNames(fnString string) ([]string, error) {
	re := regexp.MustCompile(`\(([^)]*)\)`)
	matches := re.FindStringSubmatch(fnString)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no arguments found in function string")
	}

	argNames := strings.Split(matches[1], ",")
	for i, arg := range argNames {
		argNames[i] = strings.TrimSpace(arg)
	}

	var filteredArgs []string
	for _, arg := range argNames {
		if arg != "" {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	return filteredArgs, nil
}
