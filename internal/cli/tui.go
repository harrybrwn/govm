package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/harrybrwn/govm"
	"github.com/harrybrwn/govm/internal/tui"
	"github.com/harrybrwn/x/xiter"
)

func askForInstalledVersionTUI(conf *govm.Manager, autoConfirm bool) (v govm.Version, err error) {
	logfile, err := logToFile(filepath.Join(cacheHome(), "govm-tui.log"))
	if err != nil {
		return v, err
	}
	defer logfile.Close()
	versions, err := conf.List()
	if err != nil {
		return v, err
	}
	sort.Sort(govm.VersionList(versions))
	slices.Reverse(versions)

	menu := tui.Menu[govm.Version]{
		Prompt: "Select a version:",
		OnSelect: func(index int, option *tui.MenuOption[govm.Version]) {
			v = option.Value
		},
		Options: slices.Collect(xiter.Map(xiter.Iter(versions), versionMenuOption)),
		Keys:    tui.DefaultMenuKeys(),
		Styles:  tui.DefaultMenuStyles(),
		QuitCmd: tui.NextChainedModel,
	}
	confirm := tui.Confirm{
		PromptFn: func() string {
			return fmt.Sprintf("switch to %q?", v.String())
		},
		Yes:  autoConfirm,
		Keys: tui.DefaultConfirmKeys(),
	}
	chain := tui.Chained{IgnoreProgress: true}
	chain.Models = append(chain.Models, &menu)
	if !autoConfirm {
		chain.Models = append(chain.Models, &confirm)
	}
	err = tui.Run(&chain)
	if err != nil {
		return v, err
	}
	if !menu.Selected() {
		return v, errors.New("no version selected")
	}
	if !confirm.Yes {
		return v, errors.New("cancelling version selection")
	}
	return v, nil
}

func askForDownloadableVersionTUI() (v govm.Version, err error) {
	logfile, err := logToFile(filepath.Join(cacheHome(), "govm-tui.log"))
	if err != nil {
		return v, err
	}
	defer logfile.Close()

	rawversions, err := govm.GetGoVersions()
	if err != nil {
		return v, err
	}
	versions := make([]govm.Version, len(rawversions))
	for i, rv := range rawversions {
		rv = strings.TrimPrefix(rv, "go")
		parsed, err := govm.ParseVersion(rv)
		if err != nil {
			return v, fmt.Errorf("failed to parse version %q: %w", rv, err)
		}
		versions[i] = parsed
	}
	sort.Sort(govm.VersionList(versions))
	slices.Reverse(versions)

	menu := tui.Menu[govm.Version]{
		Prompt: "Select a version:",
		OnSelect: func(index int, option *tui.MenuOption[govm.Version]) {
			v = option.Value
		},
		Options: slices.Collect(xiter.Map(xiter.Iter(versions), versionMenuOption)),
		Keys:    tui.DefaultMenuKeys(),
		Styles:  tui.DefaultMenuStyles(),
	}
	err = tui.Run(&menu)
	if err != nil {
		return v, err
	}
	if !menu.Selected() {
		return v, errors.New("no version selected")
	}
	return v, nil
}
