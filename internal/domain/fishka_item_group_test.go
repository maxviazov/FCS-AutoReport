package domain

import "testing"

func TestMoHCategoryFromFishKAGroup_arenaFish(t *testing.T) {
	got := MoHCategoryFromFishKAGroup("Arena", "ארנה - רוצעות דג")
	if got != "דגים מעובדים" {
		t.Fatalf("got %q want דגים מעובדים", got)
	}
}

func TestMoHCategoryFromFishKAGroup_empty(t *testing.T) {
	if MoHCategoryFromFishKAGroup("", "") != "" {
		t.Fatal("expected empty")
	}
}
