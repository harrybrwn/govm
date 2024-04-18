package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

//go:generate make build

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nRun 'govm help' for usage\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := NewRootCmd()
	return root.Execute()
}

const (
	defaultBase     = "/usr/local"
	rootDirName     = "go"
	versionsDirName = "govm/go-versions"
	envsDirName     = "govm/envs"
	buildCacheDir   = "govm/go-build"
	versionFilename = ".govm"
)

type Config struct {
	base           string
	rootDirName    string // should be a symlink
	versionDirName string // All the different go versions
	envsDirName    string
	currentVersion string
	versions       []string
}

func (c *Config) root() string { return filepath.Join(c.base, c.rootDirName) }

// installation returns the path of the installation for a given version
func (c *Config) installation(v string) string {
	return filepath.Join(c.base, c.versionDirName, "go"+v)
}

func (c *Config) list() (VersionList, error) {
	return list(filepath.Join(c.base, c.versionDirName))
}

func (c *Config) newest() string {
	l, err := c.list()
	if err != nil || len(l) < 1 {
		return runtime.Version() // can't find version just pick this one
	}
	return l[l.Len()-1].String()
}

// Variables that are set using ldflags
var (
	version string
	commit  string
	built   string
	docCmd  = "false"
)

func NewRootCmd() *cobra.Command {
	var (
		conf = Config{
			base:           defaultBase,
			rootDirName:    rootDirName,
			versionDirName: versionsDirName,
			envsDirName:    envsDirName,
		}
	)
	c := &cobra.Command{
		Use:           "govm",
		Short:         "Manage different versions of Go",
		Long:          "Manage different versions of Go",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return nil
		},
		Version: fmt.Sprintf("%s %s built %s", version, commit, built),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: docCmd == "true", // disable 'completion' when generating docs
		},
	}
	c.AddCommand(
		newUseCmd(&conf),
		newListCmd(&conf),
		newDownloadCmd(&conf),
		newRemoveCmd(&conf),
		newUninstallCmd(&conf),
		newEnvCmd(&conf),
	)
	if docCmd == "true" {
		c.AddCommand(newDocCmd(c))
	}
	return c
}

func newUseCmd(conf *Config) *cobra.Command {
	c := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a specified version of Go",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			versions, err := list(filepath.Join(conf.base, conf.versionDirName))
			if err != nil {
				return []string{}, cobra.ShellCompDirectiveError
			}
			versionStrings := make([]string, len(versions))
			for i, v := range versions {
				versionStrings[i] = v.String()
			}
			return versionStrings, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var toUse string
			if len(args) == 0 {
				v, err := readVersionFile(versionFilename)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf(
							"give version number or use %q file",
							versionFilename,
						)
					}
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "using version from %q\n", versionFilename)
				toUse = v.String()
			} else {
				toUse = cleanVersionInput(args[0])
			}
			return use(conf, toUse)
		},
	}
	return c
}

func newDownloadCmd(conf *Config) *cobra.Command {
	var alsoUse bool
	c := &cobra.Command{
		Use:     "download <version>",
		Short:   "Download a different version of Go",
		Aliases: []string{"dl"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := cleanVersionInput(args[0])
			err := download(conf, cmd.OutOrStdout(), v)
			if err != nil {
				return err
			}
			if alsoUse {
				return use(conf, v)
			}
			return nil
		},
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			versions, err := getGoVersions()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return versions, cobra.ShellCompDirectiveNoFileComp
		},
	}
	c.Flags().BoolVar(&alsoUse, "use", alsoUse, "set this version after downloading it")
	return c
}

func newRemoveCmd(conf *Config) *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <version>",
		Aliases: []string{"rm"},
		Short:   "Remove an installation.",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := cleanVersionInput(args[0])
			return os.RemoveAll(conf.installation(v))
		},
	}
	return c
}

func newUninstallCmd(conf *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all go versions installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstall(conf)
		},
	}
}

func newEnvCmd(conf *Config) *cobra.Command {
	return &cobra.Command{
		Use:    "env",
		Short:  "Print shell variables needed to for govm to manage your go versions.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"export GOROOT=\"%s\"\n",
				filepath.Join(conf.base, conf.rootDirName),
			)
			return nil
		},
	}
}

func use(conf *Config, version string) error {
	sym, ok := os.LookupEnv("GOROOT")
	if !ok {
		sym = filepath.Join(conf.base, conf.rootDirName)
	}
	stat, err := os.Lstat(sym)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		newest := conf.newest()
		if newest == "" {
			return errors.New("cannot find any go installations")
		}
		err = os.Symlink(
			filepath.Join(conf.base,
				conf.versionDirName,
				conf.newest(),
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
	inst := conf.installation(version)
	if !exists(inst) {
		return fmt.Errorf("version %q has not been downloaded", version)
	}
	if err = os.Remove(sym); err != nil {
		return err
	}
	fmt.Printf("switching to version %s\n", version)
	return os.Symlink(inst, sym)
}

func download(conf *Config, stdout io.Writer, version string) error {
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
	installation := conf.installation(version)
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

func uninstall(conf *Config) (err error) {
	err = os.RemoveAll(conf.root())
	if err != nil {
		return err
	}
	// Remove go installations
	err = os.RemoveAll(filepath.Join(conf.base, conf.versionDirName))
	if err != nil {
		return err
	}
	return nil
}

func newListCmd(*Config) *cobra.Command {
	c := &cobra.Command{
		Use:     "list",
		Short:   "List all the installed versions of go",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := list(filepath.Join(defaultBase, versionsDirName))
			if err != nil {
				return err
			}
			for _, name := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", name.String())
			}
			return nil
		},
	}
	return c
}

func newDocCmd(root *cobra.Command) *cobra.Command {
	var (
		stdout = false
		manDir = "release/man"
	)
	c := cobra.Command{
		Use:    "doc",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			manHead := doc.GenManHeader{
				Section: "1", // 1 is for shell commands
			}
			if stdout {
				err := doc.GenMan(root, &manHead, os.Stdout)
				if err != nil {
					return err
				}
			} else {
				if !exists(manDir) {
					_ = os.MkdirAll(manDir, 0755)
				}
				err := doc.GenManTree(root, &manHead, manDir)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&stdout, "stdout", stdout, "write docs to stdout")
	c.Flags().StringVarP(&manDir, "man-dir", "m", manDir, "directory to write man pages to")
	return &c
}

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

func readVersionFile(filename string) (*Version, error) {
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

func currentVersion(dir string) (string, error) {
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

func exists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}
