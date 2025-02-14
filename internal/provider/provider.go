package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"terraform-provider-func/internal/javascript"
	"terraform-provider-func/internal/runtime"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ssoroka/slice"
	"golang.org/x/exp/maps"
)

// Ensure FuncProvider satisfies various provider interfaces.
var _ provider.Provider = &FuncProvider{}
var _ provider.ProviderWithFunctions = &FuncProvider{}
var _ provider.ProviderWithEphemeralResources = &FuncProvider{}

// FuncProvider defines the provider implementation.
type FuncProvider struct {
	version string
	vms     map[string]runtime.Runtime
	parsed  map[string]struct{}
}

// FuncProviderModel describes the provider data model.
type FuncProviderModel struct {
	CachePath types.String `tfsdk:"cache_path"`
	Library   types.List   `tfsdk:"library"`
}

// LibraryModel describes the library data model.
type LibraryModel struct {
	Source types.String `tfsdk:"source"`
}

func (p *FuncProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Trace(ctx, "Initiating metadata")

	resp.TypeName = "func"
	resp.Version = p.version
}

func (p *FuncProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Trace(ctx, "Initiating schema")

	resp.Schema = schema.Schema{
		Description: "Bringing functional programming into Terraform.",
		MarkdownDescription: strings.Join(
			[]string{
				"The **func** provider is a powerful and unique Terraform provider that enables you to define and execute custom functions.",
				"It seamlessly blends functional and declarative paradigms, unlocking new possibilities for managing infrastructure with greater flexibility and efficiency.",
				"\nThis provider allows you to define functions in an external, functional language and then use them in your Terraform codebase.",
			},
			" ",
		),
		Attributes: map[string]schema.Attribute{
			"cache_path": schema.StringAttribute{
				Description: "Path to the local cache directory.",
				MarkdownDescription: strings.Join(
					[]string{
						"Path to the local cache directory.",
						"If not set, it defaults to `$XDG_CACHE_HOME/func/libraries`.",
						"Can also be set via an environment variable `FUNC_CACHE_PATH`.",
					},
					" ",
				),
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"library": schema.ListNestedBlock{
				MarkdownDescription: "Configuration for the functions library.",
				Description:         "Configuration for the functions library.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"source": schema.StringAttribute{
							Description: "Source of the library file.",
							MarkdownDescription: strings.Join(
								[]string{
									"Source of the library file.\n",
									"The source of the library file can be any [getter](https://github.com/hashicorp/go-getter#url-format) accepted URL (similar to Terraform module's sources).",
									"It can also be set via an environment variable like `FUNC_LIBRARY_{ID}_SOURCE`,",
									"where the `{ID}` value can be replaced with anything.",
									"The provider doesn't really care about this, as long as it is prefixed with the",
									"`FUNC_LIBRARY_` prefix, it will be found and read accordingly.",
								},
								" ",
							),
							Required: true,
						},
					},
				},
			},
		},
	}
}

func (p *FuncProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Trace(ctx, "Initiating configuration")

	var data FuncProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "could not get provider configuration", map[string]any{
			"error": formatDiagnostics(resp.Diagnostics).Error(),
		})
		return
	}

	paths, diags := FindLibrariesInModel(&data, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "could not find libraries in configuration", map[string]any{
			"error": formatDiagnostics(resp.Diagnostics).Error(),
		})
		return
	}

	for _, path := range paths {
		if _, ok := p.parsed[path]; ok {
			// This path was already parsed once.
			tflog.Debug(ctx, "library already indexed", map[string]any{
				"path": path,
			})
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			resp.Diagnostics.AddWarning("Cannot read file.", err.Error())
			tflog.Warn(ctx, "Cannot read library", map[string]any{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}

		vmKey := filepath.Ext(path)
		vm, ok := p.vms[vmKey]
		if !ok {
			resp.Diagnostics.AddWarning(
				"Cannot parse library.",
				fmt.Sprintf("There is no parser that can parse '.%s' files (source '%s').", vmKey, path),
			)
			tflog.Warn(ctx, "Cannot parse library", map[string]any{
				"path":  path,
				"vm":    vmKey,
				"error": "no VM can parse this library",
			})
			continue
		}

		if err := vm.Parse(string(content)); err != nil {
			resp.Diagnostics.AddWarning(
				"Library is unparsable.",
				fmt.Sprintf("Built-in VM could not parse library '%s': %v.", path, err.Error()),
			)
			tflog.Warn(ctx, "The vm could not parse this library", map[string]any{
				"path":  path,
				"vm":    vmKey,
				"error": err.Error(),
			})
			continue
		}

		p.parsed[path] = struct{}{}

		tflog.Info(ctx, "Successfully indexed library", map[string]any{
			"path": path,
			"vm":   vmKey,
		})
	}

	funcs := make(map[string]runtime.Function, 0)

	for _, vm := range p.vms {
		for _, f := range vm.Functions() {
			funcs[f.Name()] = f
		}
	}

	tflog.Info(ctx, "Provider indexed functions", map[string]any{
		"count": len(funcs),
		"names": maps.Keys(funcs),
	})

	resp.DataSourceData = funcs
}

func (p *FuncProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "Exposing resources")

	return []func() resource.Resource{}
}

func (p *FuncProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	tflog.Trace(ctx, "Exposing ephemeral resources")

	return []func() ephemeral.EphemeralResource{}
}

func (p *FuncProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "Exposing data sources")

	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return &DataSource{}
		},
	}
}

func (p *FuncProvider) Functions(ctx context.Context) []func() function.Function {
	tflog.Trace(ctx, "Exposing functions")

	funcs := make([]runtime.Function, 0)

	for _, runtime := range p.vms {
		funcs = append(funcs, runtime.Functions()...)
	}

	tflog.Info(ctx, "Provider indexed functions", map[string]any{
		"count": len(funcs),
		"names": slice.Map[runtime.Function, string](funcs, func(f runtime.Function) string {
			return f.Name()
		}),
	})

	return slice.Map[runtime.Function, func() function.Function](
		funcs,
		func(f runtime.Function) func() function.Function {
			return func() function.Function {
				return runtime.TerraformFunction{Function: f}
			}
		},
	)
}

func New(version string) func() provider.Provider {
	logger := newFileLogger()

	vms := map[string]runtime.Runtime{
		"js": javascript.New(),
		// "go": golang.New(),
	}

	parsed := make(map[string]struct{})

	diags := diag.Diagnostics{}

	paths, ds := FindLibrariesInEnvironment(true)
	if ds.HasError() {
		logger.Error(formatDiagnostics(ds).Error(), "diagnostics", ds)
		return nil
	}

	for _, path := range paths {
		if _, ok := parsed[path]; ok {
			// This path was already parsed once.
			logger.Debug("skipping already parsed library", "library", path)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("cannot read file", "error", err)
			diags.AddWarning("Cannot read file.", err.Error())
			continue
		}

		vmKey := strings.TrimPrefix(filepath.Ext(path), ".")
		vm, ok := vms[vmKey]
		if !ok {
			logger.Warn("no parser for file", "parser", vmKey, "path", path)
			diags.AddWarning(
				"Cannot parse library.",
				fmt.Sprintf("There is no parser that can parse '.%s' files (source '%s').", vmKey, path),
			)
			continue
		}

		if err := vm.Parse(string(content)); err != nil {
			logger.Warn("unparsable library", "parser", vmKey, "path", path, "error", err.Error())
			diags.AddWarning(
				"Library is unparsable.",
				fmt.Sprintf("Built-in VM could not parse library '%s': %v.", path, err.Error()),
			)
			continue
		}

		logger.Info("successfully parsed library", "path", path)
		parsed[path] = struct{}{}
	}

	if diags.HasError() {
		logger.Error(formatDiagnostics(ds).Error(), "diagnostics", ds)
		return nil
	}

	logger.Info("all libraries were successfully indexed", "vms", maps.Keys(vms), "parsed", maps.Keys(parsed))

	return func() provider.Provider {
		return &FuncProvider{
			version: version,
			vms:     vms,
			parsed:  parsed,
		}
	}
}

// formatDiagnostics converts a list of diagnostics into a single error.
func formatDiagnostics(ds diag.Diagnostics) error {
	for _, d := range ds {
		if d.Severity() == diag.SeverityError {
			return fmt.Errorf("%s: %s (and %d more errors)", d.Severity(), d.Detail(), ds.ErrorsCount())
		}
	}

	return fmt.Errorf("no error diagnostic")
}
