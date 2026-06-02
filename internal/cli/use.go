package cli

import (
	"fmt"

	"github.com/harrybrwn/govm"
	"github.com/spf13/cobra"
)

func newUseCmd(conf *govm.Manager) *cobra.Command {
	var noGovmFile, autoYes bool
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
			var (
				err error
				v   govm.Version
			)
			if len(args) == 0 {
				if !noGovmFile && exists(conf.VersionFile) {
					v, err = govm.ReadVersionFile(conf.VersionFile)
					if err != nil {
						return err
					}
					_, err = fmt.Fprintf(cmd.OutOrStdout(), "using version from %q\n", conf.VersionFile)
					if err != nil {
						return err
					}
				} else {
					v, err = askForInstalledVersionTUI(conf, autoYes)
					if err != nil {
						return err
					}
				}
			} else {
				v, err = govm.ParseVersion(cleanVersionInput(args[0]))
				if err != nil {
					return err
				}
			}
			err = conf.Use(v)
			if err != nil {
				return fmt.Errorf("failed to set version %q: %w", v.String(), err)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&noGovmFile, "no-govm-file", noGovmFile, "don't read the version from ./.govm")
	c.Flags().BoolVarP(&autoYes, "yes", "y", autoYes, "skip confirmation prompts")
	return c
}
