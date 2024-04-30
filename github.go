package govm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ghTag struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	Object struct {
		Sha  string `json:"sha"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"object"`
}

const (
	ghAPIHost   = "api.github.com"
	ghCacheDir  = "govm"
	ghCacheFile = "govm/golang-go-tags.json"
)

func getTagsFromGithub() ([]ghTag, error) {
	req := http.Request{
		Method: "GET",
		Host:   ghAPIHost,
		URL: &url.URL{
			Scheme: "https",
			Host:   ghAPIHost,
			Path:   "/repos/golang/go/git/refs/tags",
		},
		Header: http.Header{
			"Accept": []string{"application/vnd.github+json"},
		},
	}
	res, err := http.DefaultClient.Do(&req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	full, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	tags := make([]ghTag, 0)
	if err = json.Unmarshal(full, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func getTagsFromFileCache(filename string) ([]ghTag, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	full, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	tags := make([]ghTag, 0)
	if err = json.Unmarshal(full, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func writeGithubCache(filename string, tags []ghTag) error {
	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	raw, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	_, err = file.Write(raw)
	return err
}

func GetGoVersions() ([]string, error) {
	tmp := os.TempDir()
	cacheFile := filepath.Join(tmp, ghCacheFile)
	info, err := os.Stat(cacheFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat cache file: %w", err)
	}
	var allTags []ghTag
	if os.IsNotExist(err) || time.Since(info.ModTime()) > time.Hour {
		allTags, err = getTagsFromGithub()
		if err != nil {
			return nil, err
		}
		_ = os.Mkdir(filepath.Join(tmp, ghCacheDir), 0755)
		err = writeGithubCache(cacheFile, allTags)
		if err != nil {
			return nil, err
		}
	} else {
		allTags, err = getTagsFromFileCache(cacheFile)
		if err != nil {
			return nil, err
		}
	}
	tags := make([]string, 0, len(allTags)/2)
	for _, tag := range allTags {
		ref := strings.SplitN(tag.Ref, "/", 3)
		r := ref[2]
		if !strings.HasPrefix(r, "go") ||
			strings.Contains(r, "rc") ||
			strings.Contains(r, "beta") {
			continue
		}
		tags = append(tags, ref[2])
	}
	return tags, nil
}
