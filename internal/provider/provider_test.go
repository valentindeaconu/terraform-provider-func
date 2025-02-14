package provider

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestFuncProvider(t *testing.T) {
	// Configure the library to test
	_, filename, _, _ := runtime.Caller(0)
	dirname := filepath.Dir(filename)

	t.Setenv("FUNC_LIBRARY_TEST01_SOURCE", filepath.Join(dirname, "provider_test_library.js"))

	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_0_0), // func provider is protocol version 6
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"func": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				provider "func" {}

				data "func" "sum" {
					id = "sum"

					inputs = [100, 100]
				}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.func.sum", tfjsonpath.New("result"), knownvalue.Float64Exact(200)),
				},
			},
			{
				Config: `
				provider "func" {}

				data "func" "create_object" {
					id = "create_object"

					inputs = {
						name = "John"
						age  = 35
					}
				}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.func.create_object", tfjsonpath.New("result"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("John"),
						"age":  knownvalue.Float64Exact(35),
					})),
				},
			},
		},
	})

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0), // func provider is protocol version 6, but functions were only added in v1.8
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"func": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				output "test" {
					value = provider::func::concat("pineapple", "pen")
				}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("pineapplepen")),
				},
			},
		},
	})
}
