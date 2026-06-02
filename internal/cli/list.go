package cli

import (
	"slices"
	"sort"
	"strings"

	"github.com/harrybrwn/govm"
	"github.com/spf13/cobra"
)

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
