package domain

import "testing"

func TestAdjustCityFromFishKA_orAkiva(t *testing.T) {
	got := AdjustCityFromFishKA("אורות", "Хадера+")
	if got != "אור עקיבא" {
		t.Fatalf("got %q want אור עקיבא", got)
	}
}

func TestAdjustCityFromFishKA_orotUnchanged(t *testing.T) {
	got := AdjustCityFromFishKA("אורות", "Беэр Шева")
	if got != "אורות" {
		t.Fatalf("got %q want אורות", got)
	}
}
