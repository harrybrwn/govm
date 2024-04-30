package govm

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Manager struct {
	// Base is the base directory of all other paths.
	Base string
	// GoDir is the directory name of the Go installation relative to Base.
	GoDir         string
	VersionsDir   string
	BuildCacheDir string
	VersionFile   string
}

func NewDefaultManager() Manager {
	return Manager{
		Base:          "/usr/local",
		GoDir:         "go",
		VersionsDir:   "govm/go-versions",
		BuildCacheDir: "govm/go-build",
		VersionFile:   ".govm",
	}
}

func (m *Manager) root() string { return filepath.Join(m.Base, m.GoDir) }

func (m *Manager) installation(v string) string {
	return filepath.Join(m.Base, m.VersionsDir, "go"+v)
}

func (m *Manager) Installation(v string) string {
	return m.installation(v)
}

func (m *Manager) newest() (string, error) {
	l, err := m.List()
	if err != nil {
		return "", err
	}
	return l[l.Len()-1].String(), nil
}

func (m *Manager) List() (VersionList, error) {
	return list(filepath.Join(m.Base, m.VersionsDir))
}

func (m *Manager) Download(stdout io.Writer, version string) error {
	u, err := url.Parse(
		fmt.Sprintf(
			"https://golang.org/dl/go%s.%s-%s.tar.gz",
			version,
			runtime.GOOS,
			runtime.GOARCH,
		),
	)
	if err != nil {
		return err
	}
	t := time.Now()
	var (
		c    http.Client
		done = make(chan struct{})
	)
	defer close(done)
	go spin(done, stdout, "Downloading")
	resp, err := c.Do(&http.Request{
		Method: "GET",
		URL:    u,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("could not find version %q using %q", version, u.String())
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasSuffix(ct, "gzip") {
		return fmt.Errorf("expected a gzip response, got %s", ct)
	}

	unziped, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tarball := tar.NewReader(unziped)
	installation := m.installation(version)
	if err = os.MkdirAll(installation, 0755); err != nil && !os.IsExist(err) {
		return err
	}
	var (
		total int64
		files int64
	)
	for {
		header, err := tarball.Next()
		if err != nil {
			break
		}
		name := strings.TrimPrefix(header.Name, "go/")
		filename := filepath.Join(installation, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(filename, header.FileInfo().Mode().Perm()); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", filename, err)
			}
		case tar.TypeReg:
			dir := filepath.Dir(filename)
			if !exists(dir) {
				if err = os.MkdirAll(dir, 0775); err != nil {
					return fmt.Errorf("failed to create directory %q: %w", dir, err)
				}
			}
			f, err := os.OpenFile(
				filename,
				os.O_CREATE|os.O_WRONLY,
				header.FileInfo().Mode().Perm(),
			)
			if err != nil {
				return fmt.Errorf("failed to create regular file %q: %w", filename, err)
			}
			n, err := io.Copy(f, tarball)
			if err != nil {
				f.Close()
				return fmt.Errorf("failed to copy data to file %q: %w", filename, err)
			}
			total += n
			files++
			if err = f.Close(); err != nil {
				return fmt.Errorf("failed to close regular file %q: %w", filename, err)
			}
		default:
			return errors.New("don't know how to deal with type flag")
		}
	}
	fmt.Fprintln(stdout, "\rdownloaded", files, "files in", time.Since(t))
	fmt.Fprintln(stdout, "installed to", installation)
	return nil
}

func (m *Manager) Uninstall() (err error) {
	err = os.RemoveAll(m.root())
	if err != nil {
		return err
	}
	err = os.RemoveAll(filepath.Join(m.Base, m.VersionsDir))
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) Use(version string) error {
	sym, ok := os.LookupEnv("GOROOT")
	if !ok {
		sym = filepath.Join(m.Base, m.GoDir)
	}
	stat, err := os.Lstat(sym)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		newest, err := m.newest()
		if err != nil {
			return err
		}
		if newest == "" {
			return errors.New("cannot find any go installations")
		}
		err = os.Symlink(
			filepath.Join(m.Base,
				m.VersionsDir,
				newest,
			),
			sym,
		)
		if err != nil {
			return err
		}
	} else {
		if stat.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("%q is not a symlink, please delete it and use go%s", sym, version)
		}
	}
	inst := m.installation(version)
	if !exists(inst) {
		return fmt.Errorf("version %q has not been downloaded", version)
	}
	if err = os.Remove(sym); err != nil {
		return err
	}
	fmt.Printf("switching to version %s\n", version)
	return os.Symlink(inst, sym)
}

const loadingInterval = time.Millisecond * 250

func spin(done chan struct{}, w io.Writer, msg string) {
	var c rune
	for i := 0; ; i++ {
		select {
		case <-done:
			return
		default:
			switch i % 4 {
			case 0:
				c = '|'
			case 1:
				c = '/'
			case 2:
				c = '-'
			case 3:
				c = '\\'
			default:
				panic("modulus is broken")
			}
			fmt.Fprintf(w, "\r%s... %c", msg, c)
			time.Sleep(loadingInterval)
		}
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}
