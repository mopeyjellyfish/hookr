package memory

import (
	"math"
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
	if _, err := TryRead(mem, "field", 0, 1); err == nil ||
		!strings.Contains(err.Error(), "field") {
		t.Fatalf("expected read error, got %v", err)
	}
	if _, err := TryReadString(mem, "field", 0, 1); err == nil ||
		!strings.Contains(err.Error(), "field") {
		t.Fatalf("expected read string error, got %v", err)
	}
	if err := TryWrite(mem, "field", 0, []byte("abc")); err == nil ||
		!strings.Contains(err.Error(), "field") {
		t.Fatalf("expected write error, got %v", err)
	}
}

func TestCheckedConversions(t *testing.T) {
	if got, err := Uint32FromUint64(42); err != nil || got != 42 {
		t.Fatalf("Uint32FromUint64(42) = %d, %v", got, err)
	}
	if _, err := Uint32FromUint64(math.MaxUint32 + 1); err == nil {
		t.Fatal("expected uint64 to uint32 overflow error")
	}
	if got, err := Uint16FromUint32(7); err != nil || got != 7 {
		t.Fatalf("Uint16FromUint32(7) = %d, %v", got, err)
	}
	if _, err := Uint16FromUint32(math.MaxUint16 + 1); err == nil {
		t.Fatal("expected uint32 to uint16 overflow error")
	}
}
