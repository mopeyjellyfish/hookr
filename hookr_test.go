package hookr

import (
	"context"
	"testing"

	"github.com/mopeyjellyfish/hookr/host"
)

func TestNewPlugin(t *testing.T) {
	tests := []struct {
		name    string
		options []host.Option
		wantErr bool
	}{
		{
			name:    "valid options",
			options: []host.Option{host.WithFile("./testdata/simple/bin/simple.wasm")},
			wantErr: false,
		},
		{
			name:    "invalid options",
			options: []host.Option{host.WithFile("")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPlugin(context.Background(), tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPlugin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
