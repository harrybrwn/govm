package cli

import (
	"github.com/harrybrwn/govm"
	"github.com/spf13/cobra"
)

func newDownloadCmd(conf *govm.Manager) *cobra.Command {
	var alsoUse bool
	c := &cobra.Command{
		Use:     "download <version>",
		Short:   "Download a different version of Go",
		Aliases: []string{"dl"},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var v govm.Version
			if len(args) == 0 {
				v, err = askForDownloadableVersionTUI()
			} else {
				v, err = govm.ParseVersion(cleanVersionInput(args[0]))
			}
			if err != nil {
				return err
			}
			err = conf.Download(cmd.OutOrStdout(), v)
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
