package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/harrybrwn/govm/cmd/govm/cli"
)

func main() {
	var (
		stdout     = false
		manDir     = "release/man"
		completion = ""
	)
	flag.BoolVar(&stdout, "stdout", stdout, "write docs to stdout")
	flag.StringVar(&completion, "completion", completion, "generate completion scripts")
	flag.StringVar(&manDir, "man-dir", manDir, "directory to write man pages to")
	flag.Parse()
	root := cli.NewRootCmd()
	manHead := doc.GenManHeader{
		Section: "1", // 1 is for shell commands
	}
	if len(completion) > 0 {
		err := genComp(root, "release/completion", completion)
		if err != nil {
			log.Fatal(err)
		}
	} else if stdout {
		err := doc.GenMan(root, &manHead, os.Stdout)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if !exists(manDir) {
			_ = os.MkdirAll(manDir, 0755)
		}
		err := doc.GenManTree(root, &manHead, manDir)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func genComp(c *cobra.Command, dir, kind string) error {
	var (
		fp string
		fn func(io.Writer, bool) error
	)
	name := c.Use
	ix := strings.IndexByte(name, ' ')
	if ix > 0 {
		name = name[:ix]
	}
	switch kind {
	case "bash":
		fp = filepath.Join(dir, "bash", name)
		fn = c.GenBashCompletionV2
	case "zsh":
		fp = filepath.Join(dir, "zsh", fmt.Sprintf("_%s", name))
		fn = func(w io.Writer, _ bool) error { return c.GenZshCompletion(w) }
	case "fish":
		fp = filepath.Join(dir, "fish", fmt.Sprintf("%s.fish", name))
		fn = c.GenFishCompletion
	default:
		return fmt.Errorf("unknown completion type %q", kind)
	}
	_ = os.MkdirAll(filepath.Dir(fp), 0755)
	f, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f, true)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return !os.IsNotExist(err)
}
