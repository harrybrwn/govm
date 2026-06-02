package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harrybrwn/govm"
	"github.com/spf13/cobra"
)

type tagsListModel struct {
	list list.Model
}

func (m tagsListModel) Init() tea.Cmd { return nil }

func (m tagsListModel) View() tea.View { return tea.NewView(m.list.View()) }

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (m tagsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEscape || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// type githubTagListItem govm.GithubTag
type githubTagListItem struct {
	tag         govm.GithubTag
	description string
}

func (ght *githubTagListItem) Title() string       { return ght.tag.Ref }
func (ght *githubTagListItem) Description() string { return ght.description }
func (ght *githubTagListItem) FilterValue() string { return ght.tag.Ref }

var _ list.Item = (*githubTagListItem)(nil)

func newTestCmd(*govm.Manager) *cobra.Command {
	c := cobra.Command{
		Use:    "test",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := govm.GetGitTags()
			if err != nil {
				return err
			}
			var listItems = make([]list.Item, len(tags))
			for i, tag := range tags {
				t := strings.TrimPrefix(tag.Ref, "refs/tags/")
				listItems[i] = &githubTagListItem{
					tag: tags[i],
					description: fmt.Sprintf(
						"%s \"https://github.com/golang/go/releases/tag/%s\"",
						tag.Object.Sha,
						t,
					),
				}
			}
			model := tagsListModel{list: list.New(listItems, list.NewDefaultDelegate(), 0, 0)}
			model.list.Title = "Go Versions"
			_, err = tea.NewProgram(model).Run()
			return err
		},
	}
	return &c
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
