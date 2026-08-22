// Command plst prints the tracks contained in an M3U/M3U8 playlist,
// reading from a file argument or from stdin.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"plst/playlist"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "plst:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("plst", flag.ContinueOnError)
	dedupe := fs.Bool("dedupe", false, "drop duplicate tracks, keeping the first occurrence of each path")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: plst [-dedupe] [file.m3u | -]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	r, closeFn, err := open(fs.Args())
	if err != nil {
		return err
	}
	defer closeFn()

	list, err := playlist.Parse(r)
	if err != nil {
		return err
	}
	if *dedupe {
		list.Dedupe()
	}

	for i, t := range list.Tracks {
		title := t.Title
		if title == "" {
			title = t.Path
		}
		if t.Duration > 0 {
			fmt.Printf("%3d. %s (%s)\n", i+1, title, t.Duration.Round(time.Second))
			continue
		}
		fmt.Printf("%3d. %s\n", i+1, title)
	}
	return nil
}

// open resolves the input source: a file path argument, "-" for stdin, or
// stdin by default when no argument is given at all.
func open(args []string) (io.Reader, func() error, error) {
	if len(args) == 0 || args[0] == "-" {
		return os.Stdin, func() error { return nil }, nil
	}
	if len(args) > 1 {
		return nil, nil, fmt.Errorf("usage: plst [-dedupe] [file.m3u | -]")
	}

	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}
