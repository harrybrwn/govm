package govm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// GoDevRelease is based on https://pkg.go.dev/golang.org/x/website/internal/dl
type Release struct {
	Version string
	Stable  bool
	Files   []ReleaseFile
}

// GoDevVersionFile is based on https://pkg.go.dev/golang.org/x/website/internal/dl
type ReleaseFile struct {
	Filename       string `json:"filename"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Version        string `json:"version"`
	ChecksumSHA256 string `json:"sha256"`
	Size           int64  `json:"size"`
	Kind           string `json:"kind"` // "archive", "installer", "source"
}

func (gdvf *ReleaseFile) FullUrl() string {
	return fmt.Sprintf("https://go.dev/dl/%s", gdvf.Filename)
}

const (
	godevCacheDir            = "govm"
	godevCacheFile           = "go-dev-dl.json"
	godevCacheUpdateInterval = time.Hour * 24 * 3
)

type ReleaseOpts struct {
	StableOnly bool
}

func WithStableOnly() func(*ReleaseOpts) {
	return func(o *ReleaseOpts) { o.StableOnly = true }
}

func pullGoVersions(options ...func(*ReleaseOpts)) ([]Release, error) {
	var opts ReleaseOpts
	for _, o := range options {
		o(&opts)
	}
	tmp := os.TempDir()
	cacheDir := filepath.Join(tmp, godevCacheDir)
	cacheFile := filepath.Join(cacheDir, godevCacheFile)
	info, err := os.Stat(cacheFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var versions []Release
	if os.IsNotExist(err) || time.Since(info.ModTime()) > godevCacheUpdateInterval {
		u := url.URL{
			Scheme:   "https",
			Host:     "go.dev",
			Path:     "/dl/",
			RawQuery: "mode=json&include=all",
		}
		res, err := http.DefaultClient.Do(&http.Request{
			Method: "GET",
			Host:   u.Host,
			URL:    &u,
			Header: http.Header{
				"Accept": {"application/json"},
			},
		})
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			res.Body.Close()
			return nil, err
		}
		if err = res.Body.Close(); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &versions); err != nil {
			return nil, err
		}
		_ = os.MkdirAll(cacheDir, 0755)
		// Write to cache file
		f, err := os.OpenFile(cacheFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		if _, err = f.Write(body); err != nil {
			f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	} else {
		f, err := os.OpenFile(cacheFile, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		full, err := io.ReadAll(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(full, &versions); err != nil {
			return nil, err
		}
	}

	if opts.StableOnly {
		releases := make([]Release, 0, len(versions))
		for _, r := range versions {
			if r.Stable {
				releases = append(releases, r)
			}
		}
		return releases, nil
	}
	return versions, nil
}
