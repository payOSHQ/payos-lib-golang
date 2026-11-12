package crypto

import (
	"regexp"
	"testing"
)

func TestGenerateUUIDUniqueV4(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := GenerateUUID()
		if !re.MatchString(id) {
			t.Fatalf("invalid UUID v4: %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate UUID: %q", id)
		}
		seen[id] = struct{}{}
	}
}
