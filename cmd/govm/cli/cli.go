package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/harrybrwn/govm"
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
	return c
}

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
				fmt.Fprintf(cmd.OutOrStdout(), "using version from %q\n", conf.VersionFile)
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
	c := &cobra.Command{
		Use:     "list",
		Short:   "List all the installed versions of go",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := m.List()
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
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"export GOROOT=\"%s\"\n",
				filepath.Join(conf.Base, conf.GoDir),
			)
			return nil
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
