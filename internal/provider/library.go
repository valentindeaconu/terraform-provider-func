package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"terraform-provider-func/internal/getter"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

const (
	variablePrefix       string = "FUNC_LIBRARY_"
	sourceVariableSuffix string = "_SOURCE"
)

// getDefaultCacheFolderPath returns the default cache directory path
//
// By default, func provider stores the libraries files in the default
// user cache directory ($XDG_CACHE_HOME).
//
// If the directory does not exist, it will try to create it.
func getDefaultCacheFolderPath() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("could not find cache home")
	}

	cacheDir := filepath.Join(userCacheDir, "func", "libraries")

	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("could not create cache folder")
	}

	return cacheDir, nil
}

// FindLibrariesInEnvironment prepares libraries found in environment
// for parsing.
//
// Any found library will be downloaded and the path to the local copy
// will be returned.
//
// If optimistic is set, the search will not be canceled because some
// library cannot be processed.
func FindLibrariesInEnvironment(optimistic bool) ([]string, diag.Diagnostics) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	var fetchDst string = ""
	if path, ok := os.LookupEnv("FUNC_CACHE_PATH"); ok {
		fetchDst = path
	} else if cacheDir, err := getDefaultCacheFolderPath(); err != nil {
		diags.AddError("Cannot find default cache directory.", err.Error())
		return nil, diags
	} else {
		fetchDst = cacheDir
	}

	var appendDiag func(summary string, detail string) = diags.AddError
	if optimistic {
		appendDiag = diags.AddWarning
	}

	paths := make([]string, 0)

	for _, v := range os.Environ() {
		if strings.HasPrefix(v, variablePrefix) {
			parts := strings.SplitN(v, "=", 2)

			if len(parts) != 2 {
				// This should never happen.
				appendDiag(
					"Cannot parse environment variable.",
					fmt.Sprintf("The environment variable '%s' doesn't have the key=value format.", v),
				)
				continue
			}

			if !strings.HasSuffix(parts[0], sourceVariableSuffix) {
				// It is a func provider variable, but not the source one.
				// We don't care about it.
				continue
			}

			source := parts[1] // source of the library

			p, err := getter.Fetch(ctx, &getter.FetchInput{
				URL:  source,
				Path: fetchDst,
			})

			if err != nil {
				appendDiag("Could not download library.", err.Error())
			}

			paths = append(paths, p)
		}
	}

	return paths, nil
}

// FindLibrariesInModel prepares libraries found in a provider model.
//
// Any found library will be downloaded and the path to the local copy
// will be returned.
//
// If optimistic is set, the search will not be canceled because some
// library cannot be processed.
func FindLibrariesInModel(model *FuncProviderModel, optimistic bool) ([]string, diag.Diagnostics) {
	ctx := context.Background()
	diags := diag.Diagnostics{}

	var libs []LibraryModel = []LibraryModel{}

	diags.Append(model.Library.ElementsAs(ctx, &libs, false)...)
	if diags.HasError() {
		// It doesn't really matter if we are optimistic or not here.
		// If the conversion crashed, there is nothing to read.
		return nil, diags
	}

	if model.CachePath.IsUnknown() {
		// TODO: If the cache path is unknown, we might want to defer the execution of the library parsing.
		// But is this good, though?
		// The provider might function like this: if we know we still have some libraries to be determined
		// and parsed, we can return unknown values for any function call.
		// This might not be the best idea, since the user might use a function that is not even defined
		// in any of those "unknown" libraries.
		// We will see. For now, we cannot function without knowing where to download the files.
		diags.AddError(
			"Unknown path to the cache directory.",
			strings.Join(
				[]string{
					"The provider needs to know where to download the libraries, so the cache path cannot be unknown.",
					"Use '-target' or other options to resolve this dependency, then try again.",
				},
				" ",
			),
		)
		return nil, diags
	}

	var fetchDst string = ""
	if !model.CachePath.IsNull() && !model.CachePath.IsUnknown() {
		fetchDst = model.CachePath.ValueString()
	} else if cacheDir, err := getDefaultCacheFolderPath(); err != nil {
		diags.AddError("Cannot find default cache directory.", err.Error())
		return nil, diags
	} else {
		fetchDst = cacheDir
	}

	paths := make([]string, 0)

	var appendError func(path path.Path, summary string, detail string) = diags.AddAttributeError
	if optimistic {
		appendError = diags.AddAttributeWarning
	}

	for i, lib := range libs {
		p, err := getter.Fetch(ctx, &getter.FetchInput{
			URL:  lib.Source.ValueString(),
			Path: fetchDst,
		})
		if err != nil {
			appendError(
				path.Root("library").AtListIndex(i).AtName("source"),
				"Could not download library.",
				err.Error(),
			)

			if diags.HasError() {
				return nil, diags
			}
		}

		paths = append(paths, p)
	}

	return paths, diags
}
