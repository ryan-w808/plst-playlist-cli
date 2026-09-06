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
	"strings"
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
	write := fs.String("write", "", "write the resulting playlist to this path instead of printing a listing; \"-\" writes to stdout")
	format := fs.String("format", "", "output format for -write, \"m3u\" or \"pls\"; defaults to the -write file's extension, or m3u if that's not conclusive")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: plst [-dedupe] [-shuffle] [-seed n] [-write path] [-format m3u|pls] [file.m3u|file.pls | -]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	outPLS, err := resolveFormat(*write, *format)
	if err != nil {
		return err
	}

	r, closeFn, baseDir, isPLS, err := open(fs.Args())
	if err != nil {
		return err
	}
	defer closeFn()

	parse := playlist.Parse
	if isPLS {
		parse = playlist.ParsePLS
	}
	list, err := parse(r)
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

	if *write != "" {
		return writeList(*write, outPLS, list)
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

// resolveFormat decides which format -write should use: an explicit
// -format flag wins, then the -write path's extension, then m3u as the
// fallback since it's the more common format. format is only meaningful
// alongside -write, so it's an error to give one without the other.
func resolveFormat(write, format string) (pls bool, err error) {
	if format == "" {
		if write == "" || write == "-" {
			return false, nil
		}
		return strings.EqualFold(filepath.Ext(write), ".pls"), nil
	}
	if write == "" {
		return false, fmt.Errorf("-format requires -write")
	}
	switch strings.ToLower(format) {
	case "m3u":
		return false, nil
	case "pls":
		return true, nil
	default:
		return false, fmt.Errorf("unknown -format %q: want \"m3u\" or \"pls\"", format)
	}
}

// writeList serializes list to path in the given format, creating or
// truncating the file; path of "-" writes to stdout instead.
func writeList(path string, pls bool, list *playlist.Playlist) error {
	w := io.Writer(os.Stdout)
	if path != "-" {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	if pls {
		return playlist.WritePLS(w, list)
	}
	return playlist.Write(w, list)
}

// open resolves the input source: a file path argument, "-" for stdin, or
// stdin by default when no argument is given at all. The returned baseDir
// is the directory of the playlist file, used to resolve the relative
// track paths it contains; it's empty for stdin, which has no location of
// its own to resolve against. isPLS reports whether the file has a .pls
// extension; stdin is always treated as M3U, since there's no name to go
// by and M3U is by far the more common format to pipe around.
func open(args []string) (r io.Reader, closeFn func() error, baseDir string, isPLS bool, err error) {
	if len(args) == 0 || args[0] == "-" {
		return os.Stdin, func() error { return nil }, "", false, nil
	}
	if len(args) > 1 {
		return nil, nil, "", false, fmt.Errorf("usage: plst [-dedupe] [-shuffle] [-seed n] [-write path] [-format m3u|pls] [file.m3u|file.pls | -]")
	}

	f, err := os.Open(args[0])
	if err != nil {
		return nil, nil, "", false, err
	}
	pls := strings.EqualFold(filepath.Ext(args[0]), ".pls")
	return f, f.Close, filepath.Dir(args[0]), pls, nil
}
