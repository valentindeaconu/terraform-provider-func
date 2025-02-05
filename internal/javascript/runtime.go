package javascript

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-func/internal/runtime"

	"github.com/dop251/goja"
	"github.com/ssoroka/slice"
)

//go:embed metadata.js
var metadataScript string

// JavaScriptRuntime is a concrete implementation of the Runtime interface
// and manages a runtime for JavaScript using the goja project.
type JavaScriptRuntime struct {
	vm    *goja.Runtime
	funcs []*JavaScriptFunction
}

// New creates a new JavaScriptRuntime
func New() runtime.Runtime {
	vm := goja.New()

	return &JavaScriptRuntime{
		vm:    vm,
		funcs: make([]*JavaScriptFunction, 0),
	}
}

func (r *JavaScriptRuntime) Functions() []runtime.Function {
	return slice.Map[*JavaScriptFunction, runtime.Function](
		r.funcs,
		func(fn *JavaScriptFunction) runtime.Function {
			return fn
		},
	)
}

func (r *JavaScriptRuntime) Parse(src string) error {
	_, err := r.vm.RunString(fmt.Sprintf("%s\n\n\n%s", src, metadataScript))
	if err != nil {
		return fmt.Errorf("could not run user-defined script: %v", err)
	}

	exports := r.vm.Get("__exports").ToObject(r.vm)

	for _, key := range exports.Keys() {
		fn := exports.Get(key).ToObject(r.vm)

		name := key

		funcStr, err := r.getString(fn, "fnString", nil)
		if err != nil {
			return err
		}

		argNames, err := extractArgNames(funcStr)
		if err != nil {
			return fmt.Errorf("could no extract argument names from function %s: %w", key, err)
		}

		var rawArgsValue goja.Value
		if err := r.vm.Try(func() { rawArgsValue = fn.Get("args") }); err != nil {
			return fmt.Errorf("function %s is has an invalid 'args' declarations: %w", key, err)
		}

		var args []javaScriptArgumentInput = make([]javaScriptArgumentInput, len(argNames))
		for i, argName := range argNames {
			args[i].name = argName
			args[i].jsType = "any"
			args[i].description = ""
		}

		if rawArgsValue != nil {
			exportedRawArgs := rawArgsValue.Export()
			rawArgs, ok := exportedRawArgs.([]any)
			if !ok {
				return fmt.Errorf("function %s is has an invalid 'args' declarations: expecting array of args, but instead got %T", key, exportedRawArgs)
			}

			for i, rawArg := range rawArgs {
				if arg, ok := rawArg.(map[string]any); ok {
					if val, ok := arg["type"].(string); ok {
						args[i].jsType = val
					} else {
						return fmt.Errorf("argument %d of function %s is missing the required 'type' declaration", i, key)
					}

					if val, ok := arg["description"].(string); ok {
						args[i].description = val
					}
				} else {
					return fmt.Errorf("could not parse argument %d of function %s", i, key)
				}
			}
		}

		returnType, err := r.getString(fn, "returns", strAsPtr("any"))
		if err != nil {
			return err
		}

		funcValue, err := r.getValue(fn, "fn", true)
		if err != nil {
			return err
		}

		callable, ok := goja.AssertFunction(funcValue)
		if !ok {
			return fmt.Errorf("library exported %v func, but the JSVM could not return its pointer", name)
		}

		summary, err := r.getString(fn, "summary", strAsPtr(""))
		if err != nil {
			return err
		}

		description, err := r.getString(fn, "description", strAsPtr(""))
		if err != nil {
			return err
		}

		f, err := NewJavaScriptFunction(&javascriptFunctionInput{
			name:        name,
			summary:     summary,
			description: description,
			args:        args,
			retJsType:   returnType,
			callable:    callable,
		}, r.vm)
		if err != nil {
			return err
		}

		r.funcs = append(r.funcs, f)
	}

	return nil
}

// getValue returns an object field as a value.
func (r *JavaScriptRuntime) getValue(obj *goja.Object, field string, required bool) (goja.Value, error) {
	var val goja.Value

	if err := r.vm.Try(func() {
		val = obj.Get(field)
	}); err != nil {
		return nil, fmt.Errorf("unknown property '%s': %w", field, err)
	}

	if required && val == nil {
		return nil, fmt.Errorf("property '%s' is nil", field)
	}

	return val, nil
}

// getString returns an object field as a string.
//
// If the defaultValue is not set, the method considers the field si required
// and returns an error if it cannot find the value.
func (r *JavaScriptRuntime) getString(obj *goja.Object, field string, defaultValue *string) (string, error) {
	val, err := r.getValue(obj, field, (defaultValue == nil))
	if err != nil {
		return "", err
	}

	if val != nil {
		return val.String(), nil
	}

	return *defaultValue, nil
}

func strAsPtr(v string) *string {
	return &v
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
