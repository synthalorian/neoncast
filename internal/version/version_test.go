package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	s := String()
	if !strings.Contains(s, "neoncast") {
		t.Errorf("expected 'neoncast' in version string, got: %s", s)
	}
}

func TestShort(t *testing.T) {
	s := Short()
	if s != Version {
		t.Errorf("expected Short() to return Version, got: %s", s)
	}
}
