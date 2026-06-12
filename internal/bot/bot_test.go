package bot

import "testing"

func TestIsAllowedUser(t *testing.T) {
	if !IsAllowedUser(42, 42) {
		t.Fatal("owner must be allowed")
	}
	if IsAllowedUser(7, 42) {
		t.Fatal("non-owner must be denied")
	}
}
