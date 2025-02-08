package provider

import (
	"os"

	"github.com/hashicorp/go-hclog"
)

// newFileLogger returns a new hclog that has a dedicated
// temporary file attached, if the provider debug mode
// is enabled (via FUNC_DEBUG variable).
//
// This logger can be used to log messages outside of the
// gRPC context window (for example, during the provider
// initialization).
//
// The temporary file is created under the $TMPDIR if that
// value is set, otherwise it will be created under /tmp.
func newFileLogger() hclog.Logger {
	opts := &hclog.LoggerOptions{
		Name:  "func",
		Level: hclog.Trace,
		Color: hclog.ColorOff,
	}

	if _, ok := os.LookupEnv("FUNC_DEBUG"); ok {
		file, err := os.CreateTemp("", "terraform-provider-func-*.log")
		if err != nil {
			return hclog.New(opts)
		}

		opts.Output = file
	}

	return hclog.New(opts)
}
