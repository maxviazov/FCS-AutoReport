package store

import (
	"testing"

	"fcs-autoreport/internal/domain"
)

func TestResolveCityCodeBySubstring_industrialKannot(t *testing.T) {
	s := New()
	s.citiesMapsFromSlice([]domain.City{{Name: "כנות", Code: "J999"}})

	code, err := s.ResolveCityCodeBySubstring("איזור תעשייה כנות")
	if err != nil || code != "J999" {
		t.Fatalf("want J999, got %q %v", code, err)
	}
}

func TestResolveCityCodeBySubstring_skipsEilotNameWithWrongN61(t *testing.T) {
	s := New()
	s.citiesMapsFromSlice([]domain.City{
		{Name: "אילת", Code: "N61"},
		{Name: "אילות", Code: "N61"},
		{Name: "באר שבע", Code: "V123"},
	})

	code, err := s.ResolveCityCodeBySubstring("רחוב אילות 23")
	if err == nil && code == "N61" {
		t.Fatalf("אילות must not resolve to wrong N61, got %q", code)
	}
}
