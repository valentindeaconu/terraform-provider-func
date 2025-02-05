package runtime

// Runtime TODO
type Runtime interface {
	Parse(src string) error

	Functions() []Function
}
