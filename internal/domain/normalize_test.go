package domain

import "testing"

func TestNormalizeText_stripsBidiMarks(t *testing.T) {
	const rlm = "\u200f"
	in := rlm + "חולון" + rlm
	if got := NormalizeText(in); got != "חולון" {
		t.Fatalf("got %q want חולון", got)
	}
}

func TestCanonicalMoHCityCode(t *testing.T) {
	if got := CanonicalMoHCityCode("  n61 "); got != "N61" {
		t.Fatalf("got %q", got)
	}
}

func TestIsMoHCityCodeFormat_acceptsLowercaseLetter(t *testing.T) {
	if !IsMoHCityCodeFormat("n61") {
		t.Fatal("expected lowercase MoH code to be accepted after canonicalization")
	}
}
