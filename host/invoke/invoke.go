package invoke

import "context"

type Context struct {
	Operation string

	PluginReq  []byte
	PluginResp []byte
	PluginErr  string

	HostResp []byte
	HostErr  error
}
type invokeContextKey struct{}

func New(ctx context.Context, ic *Context) context.Context {
	return context.WithValue(ctx, invokeContextKey{}, ic)
}

func From(ctx context.Context) *Context {
	ic, _ := ctx.Value(invokeContextKey{}).(*Context)
	return ic
}
