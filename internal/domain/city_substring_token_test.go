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
	if !AllowMoHN61CityCode("אילת, רחוב 5", "", "", "") {
		t.Fatal("prefix אילת, in raw SAP address must allow N61")
	}
	if !AllowMoHN61CityCode("אילת, החורש 9", "", "", "") {
		t.Fatal("אילת, החורש 9 must allow N61")
	}
	if !AllowMoHN61CityCode("חופש באילת", "", "", "") {
		t.Fatal("באילת in address must allow N61")
	}
	if AllowMoHN61CityCode("החורש 9", "אנמגע טריד בעמ", "", "אילת") {
		t.Fatal("wrong עיר=אילת without אילת in address must not allow N61")
	}
	if AllowMoHN61CityCode("", "", "Эйлат", "") {
		t.Fatal("Russian district without אילת in address must not allow N61")
	}
	if AllowMoHN61CityCode("", "", "נפת אילת", "") {
		t.Fatal("Hebrew district without אילת in address must not allow N61")
	}
	if AllowMoHN61CityCode("", "", "דרום אילת", "") {
		t.Fatal("דרום אילת must not allow N61 (substring false positive)")
	}
}
