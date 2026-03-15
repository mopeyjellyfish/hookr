package textfilterhookr

import (
	"strings"
	"testing"
)

func TestDecodePluginInfoRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	_, err := decodePluginInfo(nil)
	if err == nil || !strings.Contains(err.Error(), "invalid flatbuffer payload") {
		t.Fatalf("expected invalid flatbuffer payload error, got %v", err)
	}
}

func TestDecodeFilterResponseRejectsShortPayload(t *testing.T) {
	t.Parallel()

	_, err := decodeFilterResponse([]byte{0x01, 0x02, 0x03})
	if err == nil || !strings.Contains(err.Error(), "invalid flatbuffer payload") {
		t.Fatalf("expected invalid flatbuffer payload error, got %v", err)
	}
}
