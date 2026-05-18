package store

import (
	"testing"

	"fcs-autoreport/internal/domain"
)

func TestGetDriverForCity_noRandomFallback(t *testing.T) {
	s := New()
	s.LoadFrom(nil, nil, []domain.Driver{
		{AgentName: "a", DriverName: "Alice", CarNumber: "111", CityCodes: "J112"},
	}, nil, nil)
	if d := s.GetDriverForCity("M37"); d != nil {
		t.Fatalf("expected nil for unmapped city, got %v", d.DriverName)
	}
	if d := s.GetDriverForCity("J112"); d == nil || d.DriverName != "Alice" {
		t.Fatalf("expected Alice for J112")
	}
}
