package client

import (
	"fmt"
	"strings"
	"testing"
)

func TestFriendlyMessageKeepsWhatACommandAdded(t *testing.T) {
	err := fmt.Errorf("%w\nuse --data for lists", &HTTPError{Status: 400, Body: "400: Bad Request"})
	if msg := FriendlyMessage(err); !strings.Contains(msg, "use --data for lists") || !strings.Contains(msg, "HTTP 400") {
		t.Fatalf("FriendlyMessage dropped the hint: %q", msg)
	}
}
