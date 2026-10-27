package syncer

import (
	"testing"

	"cloud-saver/backend/internal/models"
)

func TestProfileNeedsNewerApp(t *testing.T) {
	cases := map[int]bool{0: false, 1: false, MaxProfileSchema: false, MaxProfileSchema + 1: true}
	for schema, want := range cases {
		if got := ProfileNeedsNewerApp(models.GameProfile{SchemaVersion: schema}); got != want {
			t.Errorf("schema %d: got %v, want %v", schema, got, want)
		}
	}
}
