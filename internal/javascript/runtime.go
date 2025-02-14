package javascript

import (
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-func/internal/runtime"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/process"
	"github.com/dop251/goja_nodejs/require"
)

var (
	argNamesRegEx = regexp.MustCompile(`\(([^)]*)\)`)
)

// JavaScriptRuntime is a concrete implementation of the Runtime interface
// and manages a runtime for JavaScript using the goja project.
type JavaScriptRuntime struct {
	vm           *goja.Runtime
	funcMetadata map[string]*JavaScriptFunctionMetadata
	funcs        map[string]*JavaScriptFunction
}

// New creates a new JavaScriptRuntime.
func New() runtime.Runtime {
	vm := goja.New()

	// Enable Node.js compatibility
	_ = new(require.Registry).Enable(vm)
	process.Enable(vm)
	console.Enable(vm)

	// Create the runti,e
	runtime := &JavaScriptRuntime{
		vm:           vm,
		funcs:        make(map[string]*JavaScriptFunction, 0),
		funcMetadata: make(map[string]*JavaScriptFunctionMetadata, 0),
	}

	// Define a global function `$` that registers functions
	err := vm.Set("$", runtime.registerFn())
	if err != nil {
		panic(err)
	}

	return runtime
}

func (r *JavaScriptRuntime) Functions() []runtime.Function {
	fns := make([]runtime.Function, 0, len(r.funcs))

	for _, f := range r.funcs {
		fns = append(fns, f)
	}

	return fns
}

func (r *JavaScriptRuntime) Parse(src string) error {
	metadata, err := parseScriptJSDoc(src)
	if err != nil {
		return fmt.Errorf("cannot parse jsdoc: %w", err)
	}

	for k, v := range metadata {
		r.funcMetadata[k] = v
	}

	if _, err := r.vm.RunString(src); err != nil {
		return err
	}

	return nil
}

func (r *JavaScriptRuntime) registerFn() func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		fnRaw := call.Argument(0)

		if goja.IsUndefined(fnRaw) || goja.IsNull(fnRaw) {
			panic(r.vm.ToValue("$() requires a function: received nothing"))
		}

		fn, ok := goja.AssertFunction(fnRaw)
		if !ok {
			panic(r.vm.ToValue("$() requires a function: did not receive a function"))
		}

		fnName := fnRaw.ToObject(r.vm).Get("name").String()
		if fnName == "" {
			panic(r.vm.ToValue("Registered function must have a name"))
		}

		fnStr := fnRaw.ToObject(r.vm).String()

		f, err := r.parseFunction(fnName, fn, fnStr)
		if err != nil {
			panic(r.vm.ToValue(err))
		}

		r.funcs[fnName] = f

		return goja.Undefined()
	}
}

func (r *JavaScriptRuntime) parseFunction(name string, fn goja.Callable, fnStr string) (*JavaScriptFunction, error) {
	fnSignature := strings.SplitN(fnStr, "\n", 2)
	fnHash := removeWhitespaceFromString(fnSignature[0])

	argNames, err := extractArgNames(fnStr)
	if err != nil {
		return nil, fmt.Errorf("could no extract argument names from function %s: %w", name, err)
	}

	summary := ""
	description := ""

	var args []javaScriptArgumentInput = make([]javaScriptArgumentInput, len(argNames))
	for i, argName := range argNames {
		args[i].name = argName
		args[i].jsType = "any"
		args[i].description = ""
	}

	returnType := "any"

	metadata, ok := r.funcMetadata[fnHash]
	if ok {
		summary = metadata.summary
		description = metadata.description

		for i, param := range metadata.params {
			args[i].name = param.name
			args[i].description = param.description
			args[i].jsType = param.typ
		}

		returnType = metadata.returns.typ
	}

	return NewJavaScriptFunction(&javascriptFunctionInput{
		name:        name,
		summary:     summary,
		description: description,
		args:        args,
		retJsType:   returnType,
		callable:    fn,
	}, r.vm)
}

func extractArgNames(fnString string) ([]string, error) {
	matches := argNamesRegEx.FindStringSubmatch(fnString)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no arguments found in function signature")
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
