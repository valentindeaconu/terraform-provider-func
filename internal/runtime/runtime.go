package runtime

// Runtime is an abstract interface that represents the behavior
// of a func runtime.
type Runtime interface {
	// Parse handles the source parsing.
	//
	// A runtime can be called multiple time to parse different
	// (or even the same) sources. The runtime should handle the
	// overrides and make sure repetitive calls are allowed.
	Parse(src string) error

	// Functions returns a slice of Terraform-compatible functions.
	Functions() []Function
}
