package app

import (
	"testing"

	"fcs-autoreport/internal/domain"
)

func TestNormalizeMoHBoxColumn26(t *testing.T) {
	min := domain.MoHMinBoxesLightFraction
	if got := normalizeMoHBoxColumn26(0); got != min {
		t.Fatalf("0 -> %v, want %v", got, min)
	}
	if got := normalizeMoHBoxColumn26(0.2); got != min {
		t.Fatalf("0.2 -> %v, want %v", got, min)
	}
	if got := normalizeMoHBoxColumn26(1.5); got != 1.5 {
		t.Fatalf("1.5 -> %v, want 1.5", got)
	}
	if got := normalizeMoHBoxColumn26(min); got != min {
		t.Fatalf("threshold -> %v, want %v", got, min)
	}
}
