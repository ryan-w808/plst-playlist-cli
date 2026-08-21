// Package playlist parses audio playlist files in the M3U/M3U8 format.
package playlist

import (
	"bufio"
	"fmt"
	"io"
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
