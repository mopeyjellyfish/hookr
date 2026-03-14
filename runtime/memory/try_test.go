package memory

import (
	"strings"
	"testing"
)

func TestTryReadAndTryWrite(t *testing.T) {
	mem := &MockMemory{Data: make([]byte, 8)}
	if err := TryWrite(mem, "field", 1, []byte("abc")); err != nil {
		t.Fatalf("try write: %v", err)
	}
	buf, err := TryRead(mem, "field", 1, 3)
	if err != nil {
		t.Fatalf("try read: %v", err)
	}
	if string(buf) != "abc" {
		t.Fatalf("unexpected data: %q", string(buf))
	}
	s, err := TryReadString(mem, "field", 1, 3)
	if err != nil {
		t.Fatalf("try read string: %v", err)
	}
	if s != "abc" {
		t.Fatalf("unexpected string: %q", s)
	}
}

func TestTryReadAndTryWriteErrors(t *testing.T) {
	mem := &MockMemory{Data: make([]byte, 2), ShouldFail: true}
	if _, err := TryRead(mem, "field", 0, 1); err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("expected read error, got %v", err)
	}
	if _, err := TryReadString(mem, "field", 0, 1); err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("expected read string error, got %v", err)
	}
	if err := TryWrite(mem, "field", 0, []byte("abc")); err == nil || !strings.Contains(err.Error(), "field") {
		t.Fatalf("expected write error, got %v", err)
	}
}
