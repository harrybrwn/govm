package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/harrybrwn/govm"
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

func newUseCmd(conf *govm.Manager) *cobra.Command {
	c := &cobra.Command{
		Use:   "use <version>",
		Short: "Switch to a specified version of Go",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			versions, err := conf.List()
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
				v, err := govm.ReadVersionFile(conf.VersionFile)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf(
							"give version number or use %q file",
							conf.VersionFile,
						)
					}
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "using version from %q\n", conf.VersionFile)
				if err != nil {
					return err
				}
				toUse = v.String()
			} else {
				toUse = cleanVersionInput(args[0])
			}
			return conf.Use(toUse)
		},
	}
	return c
}

func newListCmd(m *govm.Manager) *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:     "list",
		Short:   "List all the installed versions of go",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, _ []string) (err error) {
			var (
				stdout   = cmd.OutOrStdout()
				versions []govm.Version
			)
			if all {
				vers, err := govm.GetGoVersions()
				if err != nil {
					return err
				}
				versions = make([]govm.Version, 0)
				for _, version := range vers {
					v, err := govm.ParseVersion(strings.TrimPrefix(version, "go"))
					if err != nil {
						return err
					}
					versions = append(versions, v)
				}
			} else {
				versions, err = m.List()
				if err != nil {
					return err
				}
			}
			sort.Sort(govm.VersionList(versions))
			slices.Reverse(versions)
			return listVersions(versions, stdout, noPager)
		},
	}
	c.Flags().BoolVarP(&all, "all", "a", all, "list all available versions")
	return c
}

func newDownloadCmd(conf *govm.Manager) *cobra.Command {
	var alsoUse bool
	c := &cobra.Command{
		Use:     "download <version>",
		Short:   "Download a different version of Go",
		Aliases: []string{"dl"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := cleanVersionInput(args[0])
			err := conf.Download(cmd.OutOrStdout(), v)
			if err != nil {
				return err
			}
			if alsoUse {
				return conf.Use(v)
			}
			return nil
		},
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			versions, err := govm.GetGoVersions()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return versions, cobra.ShellCompDirectiveNoFileComp
		},
	}
	c.Flags().BoolVar(&alsoUse, "use", alsoUse, "set this version after downloading it")
	return c
}

func newRemoveCmd(conf *govm.Manager) *cobra.Command {
	c := &cobra.Command{
		Use:     "remove <version>",
		Aliases: []string{"rm"},
		Short:   "Remove an installation.",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := cleanVersionInput(args[0])
			return os.RemoveAll(conf.Installation(v))
		},
	}
	return c
}

func newUninstallCmd(conf *govm.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all go versions installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			return conf.Uninstall()
		},
	}
}

func newEnvCmd(conf *govm.Manager) *cobra.Command {
	return &cobra.Command{
		Use:    "env",
		Short:  "Print shell variables needed to for govm to manage your go versions.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"export GOROOT=\"%s\"\n",
				filepath.Join(conf.Base, conf.GoDir),
			)
			return err
		},
	}
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
