package domain

import "testing"

func TestCitySubstringMatchesToken_rejectsLaEilat(t *testing.T) {
	if CitySubstringMatchesToken("רחוב לאילת 23א'", "אילת") {
		t.Fatal("לאילת must not match אילת")
	}
}

func TestCitySubstringMatchesToken_allowsBeEilat(t *testing.T) {
	if !CitySubstringMatchesToken("חופש באילת", "אילת") {
		t.Fatal("באילת must match אילת")
	}
}

func TestAllowMoHN61CityCode(t *testing.T) {
	if AllowMoHN61CityCode("נירית 101", "סופר צאלים בעמ", "", "") {
		t.Fatal("נירית must not allow N61")
	}
	if AllowMoHN61CityCode("רחוב אילות 23א'", "מעדניית דניאל בעמ", "", "") {
		t.Fatal("אילות street must not allow N61 without אילת")
	}
	if AllowMoHN61CityCode("אילת, רחוב 5", "", "", "") {
		t.Fatal("prefix אילת, alone must not allow N61 (stripped from MoH כתובת)")
	}
	if !AllowMoHN61CityCode("אילת, רחוב 5", "", "Эйлат", "") {
		t.Fatal("prefix אילת, + district Эйлат must allow N61")
	}
	if !AllowMoHN61CityCode("חופש באילת", "", "", "") {
		t.Fatal("באילת in address must allow N61")
	}
	if AllowMoHN61CityCode("החורש 9", "אנמגע טריד בעמ", "", "אילת") {
		t.Fatal("wrong עיר=אילת without אילת in address must not allow N61")
	}
	if !AllowMoHN61CityCode("", "", "Эйлат", "") {
		t.Fatal("Russian district Эйлат must allow N61")
	}
	if !AllowMoHN61CityCode("", "", "נפת אילת", "") {
		t.Fatal("Hebrew district with אילת token must allow N61")
	}
}
