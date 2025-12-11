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

func NewVersion(major, minor, patch int) Version {
	return Version{major, minor, patch, ""}
}

type Version struct {
	major int
	minor int
	patch int
	pre   string
}

func ParseVersion(str string) (v Version, err error) {
	l := strings.Split(str, ".")
	switch len(l) {
	case 3:
		v.patch, v.pre, err = parseVerNum(l[2])
		if err != nil {
			return
		}
		v.minor, err = parseInt(l[1])
		if err != nil {
			return
		}
		v.major, err = parseInt(l[0])
	case 2:
		v.minor, v.pre, err = parseVerNum(l[1])
		if err != nil {
			return
		}
		v.major, err = parseInt(l[0])
	case 1:
		v.major, v.pre, err = parseVerNum(l[0])
	default:
		return v, ErrInvalidVersion
	}
	return
}

// Cmp will compare the two sematic version numbers.
func (v *Version) Cmp(x *Version) int {
	if v.major == x.major {
		if v.minor == x.minor {
			if v.patch == x.patch {
				if v.pre == x.pre {
					return 0
				} else if v.pre < x.pre {
					return -1
				}
			} else if v.patch < x.patch {
				return -1
			}
		} else if v.minor < x.minor {
			return -1
		}
	} else if v.major < x.major {
		return -1
	}
	return 1
}

func (v *Version) String() string {
	format := "%d.%d"
	args := []any{v.major, v.minor}
	if v.patch > 0 {
		format += ".%d"
		args = append(args, v.patch)
	}
	if len(v.pre) > 0 {
		format += "%s"
		args = append(args, v.pre)
	}
	return fmt.Sprintf(format, args...)
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

func parseVerNum(s string) (int, string, error) {
	for i, c := range s {
		if c < '0' || c > '9' {
			if i == 0 {
				return 0, "", errors.New("invalid number")
			}
			n, err := strconv.ParseInt(s[:i], 10, 32)
			return int(n), s[i:], err
		}
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), "", err
}
