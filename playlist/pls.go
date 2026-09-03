package playlist

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ParsePLS reads a PLS playlist (the format used by Winamp and Shoutcast,
// and still common for internet radio streams) from r. PLS is an INI-style
// format where each track is spread across up to three keys sharing the
// same numeric suffix: FileN, TitleN, and LengthN. Those keys aren't
// required to appear in index order or grouped together, so entries are
// collected by index first and only emitted as tracks once every line has
// been read.
func ParsePLS(r io.Reader) (*Playlist, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	type entry struct {
		path     string
		title    string
		duration time.Duration
	}
	entries := make(map[int]*entry)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		idx, field, ok := splitPLSKey(key)
		if !ok {
			// NumberOfEntries, Version, and anything we don't recognize:
			// derivable from the entries themselves, so safe to skip.
			continue
		}
		e := entries[idx]
		if e == nil {
			e = &entry{}
			entries[idx] = e
		}
		switch field {
		case "file":
			e.path = value
		case "title":
			e.title = value
		case "length":
			secs, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: bad Length value %q: %w", lineNo, value, err)
			}
			// -1 means "unknown" per the PLS convention; anything else
			// non-positive isn't a real duration either.
			if secs > 0 {
				e.duration = time.Duration(secs) * time.Second
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	indexes := make([]int, 0, len(entries))
	for idx := range entries {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)

	p := &Playlist{Tracks: make([]Track, 0, len(indexes))}
	for _, idx := range indexes {
		e := entries[idx]
		if e.path == "" {
			// A TitleN or LengthN with no matching FileN isn't a track.
			continue
		}
		p.Tracks = append(p.Tracks, Track{Path: e.path, Title: e.title, Duration: e.duration})
	}
	return p, nil
}

// WritePLS serializes p to w in the PLS format.
func WritePLS(w io.Writer, p *Playlist) error {
	bw := bufio.NewWriter(w)

	if _, err := bw.WriteString("[playlist]\n"); err != nil {
		return err
	}
	for i, t := range p.Tracks {
		n := i + 1
		if _, err := fmt.Fprintf(bw, "File%d=%s\n", n, t.Path); err != nil {
			return err
		}
		title := t.Title
		if title == "" {
			title = t.Path
		}
		if _, err := fmt.Fprintf(bw, "Title%d=%s\n", n, title); err != nil {
			return err
		}
		length := int64(t.Duration.Seconds())
		if t.Duration == 0 {
			length = -1
		}
		if _, err := fmt.Fprintf(bw, "Length%d=%d\n", n, length); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(bw, "NumberOfEntries=%d\n", len(p.Tracks)); err != nil {
		return err
	}
	if _, err := bw.WriteString("Version=2\n"); err != nil {
		return err
	}
	return bw.Flush()
}

// splitPLSKey splits a PLS key like "File3" into its field name and index.
// Field names are matched case-insensitively since not every PLS writer in
// the wild capitalizes them the way the original Winamp spec does.
func splitPLSKey(key string) (idx int, field string, ok bool) {
	for _, name := range []string{"File", "Title", "Length"} {
		if len(key) <= len(name) || !strings.EqualFold(key[:len(name)], name) {
			continue
		}
		n, err := strconv.Atoi(key[len(name):])
		if err != nil || n < 1 {
			continue
		}
		return n, strings.ToLower(name), true
	}
	return 0, "", false
}
