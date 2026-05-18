package app

import "testing"

func TestSanitizeClientName_keepsBeitM(t *testing.T) {
	got := sanitizeClientName(`מינימרקט סטס בע"מ`)
	want := `מינימרקט סטס בע"מ`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
