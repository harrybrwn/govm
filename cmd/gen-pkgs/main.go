package main

import (
	"log"
	"os"

	"github.com/harrybrwn/x/nerdfont"
)

func main() {
	_ = os.MkdirAll("internal/nerdfont", 0755)
	f, err := os.OpenFile("internal/nerdfont/nerdfont.go", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	metadata, glyphs, err := nerdfont.GetGlyphs()
	if err != nil {
		log.Fatal(err)
	}
	err = nerdfont.Generate(f, &nerdfont.GenerateTemplateData{
		Package:         "nerdfont",
		Glyphs:          glyphs,
		Metadata:        *metadata,
		MappingFunction: false,
		GenList:         false,
	})
	if err != nil {
		log.Fatal(err)
	}
}
