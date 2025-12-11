package govm

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	t.Skip()
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

func TestFetchReleases(t *testing.T) {
	t.Skip()
	const url = "https://go.dev/doc/devel/release"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
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
	pat := regexp.MustCompile(`^go([0-9]+\.?){1,3}[a-z0-9]*?$`)
	for _, tag := range tags {
		if !pat.Match([]byte(tag)) {
			t.Errorf("%q is the wrong pattern", tag)
		}
	}
	for _, tag := range tags {
		v, err := ParseVersion(strings.TrimPrefix(tag, "go"))
		if err != nil {
			t.Errorf("failed to parse %q: %v", tag, err)
		}
		_ = v
	}
}

func TestVersions_GoDev(t *testing.T) {
	t.Skip()
	_ = os.Remove(filepath.Join(os.TempDir(), godevCacheDir, godevCacheFile))
	versions, err := pullGoVersions(WithStableOnly())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one version")
	}
	versions, err = pullGoVersions(WithStableOnly())
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one version")
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
		_ = os.Unsetenv(k)
	}
	return func() {
		for k, v := range values {
			_ = os.Setenv(k, v)
		}
	}
}
