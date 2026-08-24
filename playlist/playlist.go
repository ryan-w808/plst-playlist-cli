// Package playlist parses audio playlist files in the M3U/M3U8 format.
package playlist

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Track is one entry in a playlist. Title and Duration are only set when
// the source used the extended #EXTINF format; Path is always set.
type Track struct {
	Path     string
	Title    string
	Duration time.Duration
}

// Playlist is an ordered list of tracks.
type Playlist struct {
	Tracks []Track
}

// Parse reads an M3U or M3U8 playlist from r. Both the plain format (one
// path per line) and the extended format (#EXTINF header before each path)
// are accepted, and the two can be mixed within a single file.
func Parse(r io.Reader) (*Playlist, error) {
	scanner := bufio.NewScanner(r)
	// Some playlists carry very long single-line paths (e.g. streaming
	// URLs with query strings); the default 64KB scanner limit is enough
	// headroom for those without buffering the whole file.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	p := &Playlist{}
	var pending *Track
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			t, err := parseExtinf(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			pending = t
			continue
		}

		if strings.HasPrefix(line, "#") {
			// #EXTM3U header, or a directive we don't understand yet.
			continue
		}

		if pending != nil {
			pending.Path = line
			p.Tracks = append(p.Tracks, *pending)
			pending = nil
		} else {
			p.Tracks = append(p.Tracks, Track{Path: line})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return p, nil
}

// Write serializes p to w in the extended M3U format. Tracks with a Title
// or a Duration get an #EXTINF line ahead of their path; tracks with
// neither are written as a bare path, so a playlist that was parsed and
// written back out without modification round-trips line for line.
func Write(w io.Writer, p *Playlist) error {
	bw := bufio.NewWriter(w)

	if _, err := bw.WriteString("#EXTM3U\n"); err != nil {
		return err
	}
	for _, t := range p.Tracks {
		if t.Title != "" || t.Duration != 0 {
			seconds := t.Duration.Seconds()
			if _, err := fmt.Fprintf(bw, "#EXTINF:%s,%s\n", formatSeconds(seconds), t.Title); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(bw, t.Path); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// Dedupe removes tracks whose path repeats one already seen earlier in the
// playlist, keeping the first occurrence. Paths are compared after
// filepath.Clean, so "a/b.mp3" and "a/./b.mp3" count as the same track;
// they're compared as given, not resolved against a base directory, so
// two different relative paths that happen to point at the same file
// after resolution are still treated as distinct until that resolution
// step exists.
func (p *Playlist) Dedupe() {
	seen := make(map[string]bool, len(p.Tracks))
	out := p.Tracks[:0]
	for _, t := range p.Tracks {
		key := filepath.Clean(t.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	p.Tracks = out
}

// Resolve rewrites relative track paths so they're relative to baseDir
// instead of whatever directory the playlist happens to be read from.
// Callers typically pass the directory containing the playlist file, so
// that "../music/song.mp3" in the playlist resolves against the playlist's
// own location rather than the process's working directory. Paths that are
// already absolute, or that look like a URL, are left untouched.
func (p *Playlist) Resolve(baseDir string) {
	for i, t := range p.Tracks {
		if t.Path == "" || filepath.IsAbs(t.Path) || isURL(t.Path) {
			continue
		}
		p.Tracks[i].Path = filepath.Join(baseDir, t.Path)
	}
}

// isURL reports whether path looks like it starts with a URL scheme
// (e.g. "http://", "file://") rather than a filesystem path. It's a
// heuristic, not a full URL parse: good enough to avoid mangling stream
// URLs, which are common in playlists alongside local file paths.
func isURL(path string) bool {
	scheme, _, ok := strings.Cut(path, "://")
	if !ok || scheme == "" {
		return false
	}
	for _, r := range scheme {
		if r == '/' || r == '\\' {
			return false
		}
	}
	return true
}

// formatSeconds renders a duration in seconds the way EXTINF expects:
// as an integer when there's no fractional part, since that's what
// almost every M3U file in the wild uses and what parseExtinf round-trips.
func formatSeconds(seconds float64) string {
	if seconds == float64(int64(seconds)) {
		return strconv.FormatInt(int64(seconds), 10)
	}
	return strconv.FormatFloat(seconds, 'f', -1, 64)
}

// parseExtinf parses a line of the form:
//
//	#EXTINF:<seconds>,<title>
func parseExtinf(line string) (*Track, error) {
	body := strings.TrimPrefix(line, "#EXTINF:")
	comma := strings.IndexByte(body, ',')
	if comma < 0 {
		return nil, fmt.Errorf("malformed EXTINF %q", line)
	}
	secs, title := body[:comma], body[comma+1:]

	seconds, err := strconv.ParseFloat(secs, 64)
	if err != nil {
		return nil, fmt.Errorf("bad duration in EXTINF %q: %w", line, err)
	}
	if seconds < 0 {
		seconds = 0
	}
	return &Track{Title: title, Duration: time.Duration(seconds * float64(time.Second))}, nil
}
