package runtime

// HostMethodFunc is a wrapper for method-ID based host callbacks.
type HostMethodFunc struct {
	id uint32
	fn CallFn
}

// FnMethod returns the method ID and callback.
func (f HostMethodFunc) FnMethod() (id uint32, fn CallFn) {
	return f.id, f.fn
}

type HostMethod interface {
	FnMethod() (id uint32, fn CallFn)
}

// HostFnMethod creates a method-ID based host callback registration.
func HostFnMethod(id uint32, fn CallFn) HostMethodFunc {
	return HostMethodFunc{id: id, fn: fn}
}
