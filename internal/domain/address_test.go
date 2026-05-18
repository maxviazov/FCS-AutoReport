package domain

import "testing"

func TestMoHStreetLineForMoH(t *testing.T) {
	if got := MoHStreetLineForMoH("סוקולוב 63, חולון", true); got != "סוקולוב 63" {
		t.Fatalf("street,city: got %q", got)
	}
	if got := MoHStreetLineForMoH("חולון, סוקולוב 63", false); got != "סוקולוב 63" {
		t.Fatalf("city,street: got %q", got)
	}
}

func TestInferCityPlacedAfterComma(t *testing.T) {
	if !InferCityPlacedAfterComma("סוקולוב 63, חולון") {
		t.Fatal("expected suffix city")
	}
	if InferCityPlacedAfterComma("חולון, סוקולוב 63") {
		t.Fatal("expected prefix city")
	}
}

func TestNormalizeMinistryAddress_stripsPathSuffix(t *testing.T) {
	cases := []struct{ in, want string }{
		{`אשדוד, העצמאות 23\87`, "אשדוד, העצמאות 23"},
		{`העצמאות 23\87`, "העצמאות 23"},
		{`העצמאות 23\\87`, "העצמאות 23"},
		{`שבי ציון 2/124`, "שבי ציון 2"},
		{`ראשונים 26/113`, "ראשונים 26"},
		{"רחוב 5", "רחוב 5"},
	}
	for _, tc := range cases {
		if got := NormalizeMinistryAddress(tc.in); got != tc.want {
			t.Fatalf("NormalizeMinistryAddress(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMoHStreetWithoutLeadingCity(t *testing.T) {
	cases := []struct{ in, want string }{
		{"תל אביב, שוק תקווה 39", "שוק תקווה 39"},
		{"אילת, החורש 9", "החורש 9"},
		{"קציר-חריש, דרך ארץ 76", "דרך ארץ 76"},
		{"רחוב ללא עיר", "רחוב ללא עיר"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := MoHStreetWithoutLeadingCity(tc.in); got != tc.want {
			t.Fatalf("MoHStreetWithoutLeadingCity(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
