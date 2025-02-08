# Terraform Provider Func

The **func** provider is a powerful and unique Terraform provider that enables you to define and execute custom functions. It seamlessly blends functional and declarative paradigms, unlocking new possibilities for managing infrastructure with greater flexibility and efficiency.

In essence, the **func** provider allows you to build functional libraries that integrate natively into your Terraform codebase. These libraries are evaluated at runtime, eliminating the need to develop an entire provider just to introduce a small piece of functionality. With **func**, you can extend Terraform dynamically, keeping your infrastructure code clean, modular, and adaptable.

## Proof of concept

For now, the provider is still in the state of the art phase.

Features - completed, work-in-progress or planned (in no specific order):
- [x] Dynamic functions generation;
- [x] JavaScript support (via goja);
- [x] Getter integration;
- [x] JSDoc integration;
- [ ] GoLang support;
- [ ] Provider configuration;
- [x] Terraform <1.8 support via data-sources.

## Features

### Libraries alongside codebase

You can now define libraries of functions alongside your infrastructure code.

1. Create a new JavaScript file `lib.js`.
2. Place it right next to your Terraform files. 
3. Open it and write your own function:
   ```javascript
    /**
     * Check if a string includes a substring.
     * 
     * This function checks if a given substring is part of a string.
     * @param {string} s the string
     * @param {string} sub the substring
     * @returns {boolean} whether the string contains the substring or not
     */
    $(function string_includes(s, sub) {
      return s.includes(sub);
    })
    ```
4. Configure the provider:
    ```hcl
    terraform {
      required_providers {
        func = {
          source = "valentindeaconu/func"
        }
      }
    }

    # This is not yet supported by Terraform.
    # You can achieve this by setting the  environment variable.
    provider "func" {
      # You can either declare the library as code here,
      # or set it via an environment variable: FUNC_LIBRARY_{ID}_SOURCE="file:///abs/path/to/the/lib.js"
      library {
        source = "file://${path.module}/lib.js"
      }
    }
    ```
5. Use the function:
   ```hcl
   user_message = provider::func::string_includes("Hello, world!", "world") ? "This is cool" : "Not so much"
   ```

The func provider will look up for all environment variables that have the `FUNC_` prefix and will use them to auto-configure itself. You can add any number of sources you would like using the environment variables, by simply changing the `ID` value in the variable name (e.g. `FUNC_LIBRARY_0001_SOURCE`, `FUNC_LIBRARY_0002_SOURCE`). Those IDs are not neither stored nor used internally, so you can name them anything you like. Their only purpose is to differentiate between sources. Keep in mind that the order of their parsing is not defined by the func provider, so try to avoid overriding functions.

### Remote libraries

The func provider integrates with go-getter, so you can fetch your libraries at runtime from any remote source, using the exact same sources you will provide for your modules.

### Terraform Language-server support

By annotating your functions with JSDoc descriptions, the func provider will gather those comments and communicate them to the language-server, so you can see what you are doing directly from your IDE.

### Multiple runtimes

Depending on your library file extension, you can either use JavaScript or GoLang (planned) to declare your functions. The provider will handle the interpretation under the hood. 

### Data sources

If you are on a Terraform version lower than 1.8, don't worry - you can still use the func provider via data-sources!

```hcl
data "func" "sum_numbers" {
  # The name of the function you want to use
  id = "sum"

  # Either a tuple with the inputs
  inputs = [10, 32]

  # Or, an object with named parameters
  inputs = {
    a = 10
    b = 32
  }
}
```

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Using the provider

```hcl
terraform {
  required_providers {
    func = {
      source = "valentindeaconu/func"
    }
  }
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
