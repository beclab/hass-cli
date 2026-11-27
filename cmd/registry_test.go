package cmd

import (
	"strings"
	"testing"
)

func TestRegistryWireNames(t *testing.T) {
	if got := registryWSOp("entity", "delete"); got != "remove" {
		t.Errorf("entity delete sends %q, want remove", got)
	}
	if got := registryWSOp("area", "delete"); got != "delete" {
		t.Errorf("area delete sends %q, want delete", got)
	}
	if got := registryListCols["device"]; !strings.HasPrefix(got, "ID=id,") {
		t.Errorf("device list reads its id from %q; devices use id", got)
	}
}
