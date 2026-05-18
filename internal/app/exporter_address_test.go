package app

import "testing"

func TestMohStreetFromPostExportAddress(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantOK  bool
	}{
		{"סוקולוב 63, חולון", "סוקולוב 63", true},
		{"קרן היסוד, 68", "", false},
		{"הרצל 30", "", false},
		{"חולון, סוקולוב 63", "", false},
	}
	for _, tc := range cases {
		got, ok := mohStreetFromPostExportAddress(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Fatalf("mohStreetFromPostExportAddress(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}
