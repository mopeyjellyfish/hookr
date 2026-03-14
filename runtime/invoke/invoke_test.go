package invoke

import (
	"context"
	"testing"
)

func TestNewAndFrom(t *testing.T) {
	base := context.Background()
	ic := &Context{Operation: "test"}
	ctx := New(base, ic)
	if got := From(ctx); got != ic {
		t.Fatalf("unexpected invoke context: %#v", got)
	}
	if got := From(base); got != nil {
		t.Fatalf("expected nil from base context, got %#v", got)
	}
}
