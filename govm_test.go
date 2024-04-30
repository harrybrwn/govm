package govm

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

func TestUse(t *testing.T) {
	cleanup := resetEnv()
	defer cleanup()
	m := Manager{
		Base:        t.TempDir(),
		GoDir:       "go",
		VersionsDir: "govm/go-versions",
	}
	setup(&m, t)
	version := "1.0"
	_ = os.Mkdir(m.installation(version), 0755)
	sym := filepath.Join(m.Base, m.GoDir)
	err := m.Use(version)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(sym)
	if err != nil {
		t.Fatalf("could not stat file %q: %v", sym, err)
	}
	if stat.Mode()&fs.ModeSymlink != 0 {
		t.Errorf("expected %q to be a symlink", sym)
	}
}

func TestDownload(t *testing.T) {
	cleanup := resetEnv()
	defer cleanup()
	m := Manager{
		Base:        t.TempDir(),
		GoDir:       "go",
		VersionsDir: "govm/go-versions",
	}
	setup(&m, t)
	version := "1.22.0"
	err := m.Download(io.Discard, version)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"VERSION", "LICENSE", "README.md", "lib", "api", "pkg"} {
		f := filepath.Join(m.installation(version), name)
		if !exists(f) {
			t.Errorf("expected %q to exist", f)
		}
	}
}

func TestValidateSemvar(t *testing.T) {
	t.Run("TestValidateSemvar_Ok", func(t *testing.T) {
		for _, v := range []string{
			"v0.0",
			"1.16",
			"1.17.3",
			"go9.8.9999",
			"49.0",
			"1.1.1",
			"v1.1",
		} {
			err := validateVersion(v)
			if err != nil {
				t.Errorf("failed validate %q: %v", v, err)
			}
		}
	})
	t.Run("TestValidateSemvar_Err", func(t *testing.T) {
		for _, v := range []string{
			"bad",
			"1",
			"49.",
			"-0.4.9",
			"1.1.1.1",
		} {
			err := validateVersion(v)
			if err == nil {
				t.Errorf("expected error for %q", v)
			}
		}
	})
}

func TestParseVersion(t *testing.T) {
	type table struct {
		in  string
		exp Version
	}
	for _, tt := range []table{
		{"4.5.6", Version{4, 5, 6}},
		{"100.9", Version{100, 9, 0}},
		{"0.1.2", Version{0, 1, 2}},
		{"9.33", Version{9, 33, 0}},
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
	for _, s := range []string{
		"one.two.four",
		"2",
		"9.8.4.6",
		"1.2.three",
		"1.two.3",
		"one.2.3",
	} {
		_, err := ParseVersion(s)
		if err == nil {
			t.Errorf("expected error when parsing %q", s)
		}
	}
}

func TestVersion_Cmp(t *testing.T) {
	if (&Version{1, 18, 0}).Cmp(&Version{1, 17, 0}) <= 0 {
		t.Fatal("should be greater than")
	}
	for _, v := range []Version{
		{1, 1, 1},
		{1, 90, 6},
		{8, 17, 4},
	} {
		v1 := Version{v.major, v.minor, v.patch}
		if v.Cmp(&v1) != 0 {
			t.Fatalf("%v should equal %v", v1, v)
		}
	}
	base := Version{2, 18, 5}
	for _, v := range []Version{
		{2, 18, 6},
		{2, 19, 100},
		{2, 19, 0},
		{3, 18, 5},
	} {
		if base.Cmp(&v) >= 0 {
			t.Errorf("%v should be less than %v", v, base)
		}
	}
}

func TestVersionList(t *testing.T) {
	vl := VersionList{
		{1, 18, 0},
		{1, 17, 5},
		{1, 11, 0},
		{1, 18, 5},
		{1, 17, 3},
		{1, 19, 0},
		{1, 16, 10},
		{1, 19, 3},
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
}

func TestFetchReleases(t *testing.T) {
	t.Skip()
	const url = "https://go.dev/doc/devel/release"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTagsFromGH(t *testing.T) {
	tags, err := GetGoVersions()
	if err != nil {
		t.Fatal(err)
	}
	pat := regexp.MustCompile(`^go([0-9]+\.?){1,3}$`)
	for _, tag := range tags {
		if !pat.Match([]byte(tag)) {
			t.Errorf("%q is the wrong pattern", tag)
		}
	}
}

func setup(conf *Manager, t *testing.T) {
	t.Helper()
	conf.Base = t.TempDir()
	err := os.MkdirAll(filepath.Join(conf.Base, conf.VersionsDir), 0775)
	if err != nil {
		t.Error(err)
	}
}

func resetEnv() func() {
	values := map[string]string{
		"GOROOT": "",
	}
	for k := range values {
		values[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range values {
			os.Setenv(k, v)
		}
	}
}
