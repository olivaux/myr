// cmd/mangen/main.go — generate man pages for the myr CLI from Cobra commands.
//
// Usage:
//
//	go run ./cmd/mangen           # generates into docs/man/man1/
//	go run ./cmd/mangen -o /tmp   # generates into /tmp/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra/doc"
	"myr-core/adapters/in/cli"
)

func main() {
	outDir := flag.String("o", "docs/man/man1", "output directory for .1 files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("cannot create %s: %v", *outDir, err)
	}

	header := &doc.GenManHeader{
		Title:   "MYR",
		Section: "1",
		Date:    func() *time.Time { t := time.Now(); return &t }(),
		Source:  "Myr Project",
		Manual:  "Myr Admin CLI — 3D Model Platform on Blockchain",
	}

	root := cli.CommandTree()
	// Disable the default cobra completion command from appearing in man pages.
	root.CompletionOptions.DisableDefaultCmd = true

	if err := doc.GenManTree(root, header, *outDir); err != nil {
		log.Fatalf("man page generation failed: %v", err)
	}

	entries, _ := os.ReadDir(*outDir)
	fmt.Printf("Man pages written to %s/\n", *outDir)
	for _, e := range entries {
		fmt.Printf("  %s\n", e.Name())
	}
}
