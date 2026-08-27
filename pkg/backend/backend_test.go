package backend

import (
	"testing"
	"toru/pkg/backend/nix"
	"toru/pkg/backend/yay"
)

func TestBackendRegistry(t *testing.T) {
	Register(nix.New())
	Register(yay.New())

	_, err := Get("nix")
	if err != nil {
		t.Fatalf("expected nix backend to be registered, got error: %v", err)
	}

	_, err = Get("yay")
	if err != nil {
		t.Fatalf("expected yay backend to be registered, got error: %v", err)
	}

	_, err = Get("unknown")
	if err == nil {
		t.Fatalf("expected error for unknown backend, got nil")
	}
}
