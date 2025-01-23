// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"terraform-provider-func/internal/functions"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ssoroka/slice"
)

// Ensure FuncProvider satisfies various provider interfaces.
var _ provider.Provider = &FuncProvider{}
var _ provider.ProviderWithFunctions = &FuncProvider{}
var _ provider.ProviderWithEphemeralResources = &FuncProvider{}

// FuncProvider defines the provider implementation.
type FuncProvider struct {
	version string
	parser  *functions.Parser
}

// FuncProviderModel describes the provider data model.
type FuncProviderModel struct {
	Library types.List `tfsdk:"library"`
}

// LibraryModel describes the library data model.
type LibraryModel struct {
	File types.String `tfsdk:"file"`
}

func (p *FuncProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "func"
	resp.Version = p.version
}

func (p *FuncProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"library": schema.ListNestedBlock{
				MarkdownDescription: "Configuration for the functions library.",
				Description:         "Configuration for the functions library.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"file": schema.StringAttribute{
							Description:         "Path to the JavaScript file.",
							MarkdownDescription: "Path to the JavaScript file.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (p *FuncProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data FuncProviderModel
	var libs []LibraryModel = []LibraryModel{}

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Library.ElementsAs(ctx, libs, false)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning("found library files", fmt.Sprintf("%v %T", libs, libs))

	for _, lib := range libs {
		resp.Diagnostics.AddWarning("parsing library file", fmt.Sprintf("%v %T", lib, lib))

		content, err := os.ReadFile(lib.File.String())
		if err != nil {
			resp.Diagnostics.AddError("could not read library file", err.Error())
			return
		}

		p.parser.Parse(string(content))
	}
}

func (p *FuncProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *FuncProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *FuncProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *FuncProvider) Functions(ctx context.Context) []func() function.Function {
	return slice.Map[functions.JSFunction, func() function.Function](
		p.parser.GetFunctions(),
		func(f functions.JSFunction) func() function.Function {
			return func() function.Function {
				return f
			}
		},
	)
}

func New(version string) func() provider.Provider {
	parser := functions.New()

	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "TF_PROVIDER_FUNC_LIBRARY") {
			file := strings.SplitN(v, "=", 2)
			content, err := os.ReadFile(file[1])
			if err != nil {
				tflog.Error(context.TODO(), fmt.Sprintf("ignored file %v: %w", file, err))
				continue
			}

			parser.Parse(string(content))
		}
	}

	return func() provider.Provider {
		return &FuncProvider{
			version: version,
			parser:  parser,
		}
	}
}
