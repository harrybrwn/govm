package govm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

var ErrInvalidVersion = errors.New("invalid version")

type Version struct {
	major int
	minor int
	patch int
}

func ParseVersion(str string) (v Version, err error) {
	l := strings.Split(str, ".")
	switch len(l) {
	case 3:
		v.patch, err = parseInt(l[2])
		if err != nil {
			return
		}
		fallthrough
	case 2:
		v.major, err = parseInt(l[0])
		if err != nil {
			return
		}
		v.minor, err = parseInt(l[1])
		if err != nil {
			return
		}
		return v, err
	default:
		return v, ErrInvalidVersion
	}
}

// Cmp will compare the two sematic version numbers.
func (v *Version) Cmp(x *Version) int {
	if v.major == x.major {
		if v.minor == x.minor {
			if v.patch == x.patch {
				return 0
			} else if v.patch < x.patch {
				return -1
			}
			return 1
		} else if v.minor < x.minor {
			return -1
		}
		return 1
	} else if v.major < x.major {
		return -1
	}
	return 1
}

func (v *Version) String() string {
	if v.patch > 0 {
		return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	}
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

// VersionList is a sortable list of semantic version numbers.
type VersionList []Version

func (vl VersionList) Len() int { return len(vl) }

func (vl VersionList) Less(i, j int) bool {
	return vl[i].Cmp(&vl[j]) < 0
}

func (vl VersionList) Swap(i, j int) { vl[i], vl[j] = vl[j], vl[i] }

func list(dir string) (VersionList, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	versions := make(VersionList, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimPrefix(e.Name(), "go")
		v, err := ParseVersion(name)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	sort.Sort(versions)
	return versions, nil
}

func ReadVersionFile(filename string) (*Version, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	raw = bytes.Trim(raw, " \r\n\t")
	if bytes.HasPrefix(raw, []byte("go")) {
		raw = raw[2:]
	}
	v, err := ParseVersion(string(raw))
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func CurrentVersion(dir string) (string, error) {
	stat, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}
	if stat.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("not a symlink")
	}
	return os.Readlink(dir)
}

type tag struct {
	Name    string `json:"name"`
	Zipball string `json:"zipball_url"`
	Tarball string `json:"tarball_url"`
	Commit  struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	NodeID string `json:"node_id"`
}

func cleanVersionInput(in string) string {
	if in[0] == 'v' {
		in = in[1:]
	}
	in = strings.TrimPrefix(in, "go")
	return in
}

func parseInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}
