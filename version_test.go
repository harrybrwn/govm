package govm

import (
	"sort"
	"testing"
)

func TestParseVersion(t *testing.T) {
	type table struct {
		in  string
		exp Version
	}
	t.Run("Success", func(t *testing.T) {
		for _, tt := range []table{
			{"4.5.6", Version{4, 5, 6, ""}},
			{"100.9rc", Version{100, 9, 0, "rc"}},
			{"0.1.2", Version{0, 1, 2, ""}},
			{"9.33beta", Version{9, 33, 0, "beta"}},
		} {
			v, err := ParseVersion(tt.in)
			if err != nil {
				t.Error(err)
				continue
			}
			if tt.exp.Cmp(&v) != 0 {
				t.Errorf("error parsing %q: %v != %v", tt.in, v, tt.exp)
				continue
			}
		}
	})
	t.Run("Failures", func(t *testing.T) {
		for _, s := range []string{
			"one.two.four",
			"9.8.4.6",
			"1.2.three",
			"1.two.3",
			"one.2.3",
			"9.bad",
		} {
			_, err := ParseVersion(s)
			if err == nil {
				t.Errorf("expected error when parsing %q", s)
			}
		}
	})
}

func TestVersion_Cmp(t *testing.T) {
	if (&Version{1, 18, 0, ""}).Cmp(&Version{1, 17, 0, ""}) <= 0 {
		t.Fatal("should be greater than")
	}
	for _, v := range []Version{
		{1, 1, 1, ""},
		{1, 90, 6, ""},
		{8, 17, 4, ""},
	} {
		v1 := Version{v.major, v.minor, v.patch, ""}
		if v.Cmp(&v1) != 0 {
			t.Fatalf("%v should equal %v", v1, v)
		}
	}
	base := Version{2, 18, 5, ""}
	for _, v := range []Version{
		{2, 18, 6, ""},
		{2, 19, 100, ""},
		{2, 19, 0, ""},
		{3, 18, 5, ""},
	} {
		if base.Cmp(&v) >= 0 {
			t.Errorf("%v should be less than %v", v, base)
		}
	}
}

func TestVersionList(t *testing.T) {
	vl := VersionList{
		NewVersion(1, 18, 0),
		{1, 17, 5, ""},
		{1, 11, 0, ""},
		{1, 18, 5, ""},
		{1, 17, 3, ""},
		{1, 19, 0, ""},
		{1, 16, 10, ""},
		{1, 19, 3, ""},
	}
	if vl.Len() != len(vl) {
		t.Fatal("VersionList.Len should equal len")
	}
	if vl.Less(0, 1) {
		t.Fatal("greater version number should not be marked as less")
	}
	if !vl.Less(1, 0) {
		t.Fatal("less version number should be marked as less")
	}
	vl.Swap(0, 1)
	if vl.Less(1, 0) {
		t.Fatal("greater version number should not be marked as less")
	}
	if !vl.Less(0, 1) {
		t.Fatal("less version number should be marked as less")
	}
	sort.Sort(vl)
	for i := 1; i < len(vl); i++ {
		if vl[i-1].Cmp(&vl[i]) > 0 {
			t.Errorf("%v should be less than %v after sorting", vl[i-1], vl[i])
		}
	}
	vl = VersionList{
		{1, 2, 3, "rc"},
		{1, 2, 3, ""},
	}
	sort.Sort(vl)
	if vl[0].pre != "" {
		t.Error("version list was incorrectly sorted")
	}
	if vl[1].pre != "rc" {
		t.Error("version list was incorrectly sorted")
	}
	if vl[0].String() != "1.2.3" {
		t.Error("incorrect version string")
	}
	if vl[1].String() != "1.2.3rc" {
		t.Error("incorrect version string")
	}
}
