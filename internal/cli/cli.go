// Package cli holds the cli commands.
package cli

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/harrybrwn/govm"
	"github.com/harrybrwn/govm/internal/tui"
	"github.com/harrybrwn/x/cobrautil"
	"github.com/harrybrwn/x/stdio"
)

// Variables that are set using ldflags
var (
	version    string
	commit     string
	built      string
	completion = "false"
)

func NewRootCmd() *cobra.Command {
	var conf = govm.NewDefaultManager()
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
			DisableDefaultCmd: completion != "true", // disable 'completion' when generating docs
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
	c.SetUsageTemplate(cobrautil.IndentedCobraUsageTemplate)
	flags := c.PersistentFlags()
	flags.BoolVar(&noPager, "no-pager", noPager, "disable automatic paging with $PAGER or $GOVM_PAGER")
	flags.BoolVar(&noCache, "no-cache", noCache, "disable caching")
	_ = flags.MarkHidden("no-cache")
	return c
}

var (
	noCache bool
	noPager bool
)

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func newRemoveCmd(conf *govm.Manager) *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <version>",
		Aliases: []string{"rm"},
		Short:   "Remove an installation.",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := govm.ParseVersion(cleanVersionInput(args[0]))
			if err != nil {
				return err
			}
			return os.RemoveAll(conf.Installation(v))
		},
	}
	return c
}

func cleanVersionInput(in string) string {
	if in[0] == 'v' {
		in = in[1:]
	}
	in = strings.TrimPrefix(in, "go")
	return in
}

func listVersions(versions []govm.Version, stdout io.Writer, noPager bool) error {
	pager := stdio.FindPager("GOVM_PAGER")
	_, height, err := term.GetSize(0)
	if err != nil {
		return err
	}
	if len(versions) > height && len(pager) > 0 && !noPager {
		var b bytes.Buffer
		for _, v := range versions {
			_, err = fmt.Fprintf(&b, "%s\n", v.String())
			if err != nil {
				return err
			}
		}
		return stdio.Page(pager, stdout, &b)
	} else {
		for _, v := range versions {
			_, err = fmt.Fprintf(stdout, "%s\n", v.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func logToFile(filename string) (io.Closer, error) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	h := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(h))
	return f, nil
}

func versionMenuOption(v govm.Version) tui.MenuOption[govm.Version] {
	s := v.String()
	return tui.MenuOption[govm.Version]{
		ID:      s,
		Display: fmt.Sprintf("v%s", s),
		Value:   v,
	}
}

func cacheHome() string {
	v, ok := os.LookupEnv("XDG_CACHE_HOME")
	if ok {
		return v
	}
	v, ok = os.LookupEnv("HOME")
	if ok {
		return filepath.Join(v, ".cache")
	}
	return os.TempDir()
}
