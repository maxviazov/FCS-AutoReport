package domain

import "testing"

func TestHebrewCityHintFromDistrictLabel(t *testing.T) {
	h, ok := HebrewCityHintFromDistrictLabel("Холон")
	if !ok || h != "חולון" {
		t.Fatalf("Холон -> %q ok=%v", h, ok)
	}
	h, ok = HebrewCityHintFromDistrictLabel("Эйлат")
	if !ok || h != "אילת" {
		t.Fatalf("Эйлат -> %q ok=%v", h, ok)
	}
	if _, ok := HebrewCityHintFromDistrictLabel("חולון"); ok {
		t.Fatal("Hebrew should not map")
	}
}
