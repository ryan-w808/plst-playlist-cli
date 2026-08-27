// Command plst prints the tracks contained in an M3U/M3U8 playlist,
// reading from a file argument or from stdin.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
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
	shuffle := fs.Bool("shuffle", false, "randomize track order")
	seed := fs.Int64("seed", 0, "seed for -shuffle, for a reproducible order; 0 picks a new random seed each run")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: plst [-dedupe] [-shuffle] [-seed n] [file.m3u | -]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	r, closeFn, baseDir, err := open(fs.Args())
	if err != nil {
		return err
	}
	defer closeFn()

	list, err := playlist.Parse(r)
	if err != nil {
		return err
	}
	if baseDir != "" {
		list.Resolve(baseDir)
	}
	if *dedupe {
		list.Dedupe()
	}
	if *shuffle {
		s := *seed
		if s == 0 {
			s = time.Now().UnixNano()
		}
		list.Shuffle(rand.New(rand.NewSource(s)))
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
// stdin by default when no argument is given at all. The returned baseDir
// is the directory of the playlist file, used to resolve the relative
// track paths it contains; it's empty for stdin, which has no location of
// its own to resolve against.
func open(args []string) (r io.Reader, closeFn func() error, baseDir string, err error) {
	if len(args) == 0 || args[0] == "-" {
		return os.Stdin, func() error { return nil }, "", nil
	}
	if len(args) > 1 {
		return nil, nil, "", fmt.Errorf("usage: plst [-dedupe] [-shuffle] [-seed n] [file.m3u | -]")
	}

	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, "", err
	}
	return f, f.Close, filepath.Dir(args[0]), nil
}
