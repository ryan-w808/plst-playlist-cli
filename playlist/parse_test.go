package playlist

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParsePlain(t *testing.T) {
	input := "song1.mp3\nsong2.mp3\n"
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"song1.mp3", "song2.mp3"}
	if len(p.Tracks) != len(want) {
		t.Fatalf("got %d tracks, want %d: %+v", len(p.Tracks), len(want), p.Tracks)
	}
	for i, path := range want {
		if p.Tracks[i].Path != path {
			t.Errorf("track %d: got path %q, want %q", i, p.Tracks[i].Path, path)
		}
		if p.Tracks[i].Title != "" || p.Tracks[i].Duration != 0 {
			t.Errorf("track %d: got %+v, want no title or duration", i, p.Tracks[i])
		}
	}
}

func TestParseExtended(t *testing.T) {
	input := `#EXTM3U
#EXTINF:123,Artist - Title
song1.mp3
#EXTINF:200,Another Track
song2.mp3
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []Track{
		{Path: "song1.mp3", Title: "Artist - Title", Duration: 123 * time.Second},
		{Path: "song2.mp3", Title: "Another Track", Duration: 200 * time.Second},
	}
	if len(p.Tracks) != len(want) {
		t.Fatalf("got %d tracks, want %d: %+v", len(p.Tracks), len(want), p.Tracks)
	}
	for i, tr := range want {
		if p.Tracks[i] != tr {
			t.Errorf("track %d: got %+v, want %+v", i, p.Tracks[i], tr)
		}
	}
}

func TestParseMixedPlainAndExtended(t *testing.T) {
	input := `#EXTINF:60,Has Metadata
song1.mp3
song2.mp3
`
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Tracks) != 2 {
		t.Fatalf("got %d tracks, want 2: %+v", len(p.Tracks), p.Tracks)
	}
	if p.Tracks[0].Title != "Has Metadata" || p.Tracks[0].Duration != 60*time.Second {
		t.Errorf("track 0: got %+v, want metadata from EXTINF", p.Tracks[0])
	}
	if p.Tracks[1].Path != "song2.mp3" || p.Tracks[1].Title != "" {
		t.Errorf("track 1: got %+v, want bare path with no title", p.Tracks[1])
	}
}

func TestParseFractionalSeconds(t *testing.T) {
	p, err := Parse(strings.NewReader("#EXTINF:12.5,Fraction\nsong.mp3\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := time.Duration(12.5 * float64(time.Second))
	if p.Tracks[0].Duration != want {
		t.Errorf("got duration %v, want %v", p.Tracks[0].Duration, want)
	}
}

func TestParseExtinfNegativeDuration(t *testing.T) {
	// -1 shows up in the wild for "unknown length"; it should clamp to 0
	// rather than producing a negative time.Duration.
	p, err := Parse(strings.NewReader("#EXTINF:-1,Unknown Length\nsong.mp3\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Tracks[0].Duration != 0 {
		t.Errorf("got duration %v, want 0 for negative EXTINF duration", p.Tracks[0].Duration)
	}
}

func TestParseExtinfMissingComma(t *testing.T) {
	_, err := Parse(strings.NewReader("#EXTINF:123 no comma here\nsong.mp3\n"))
	if err == nil {
		t.Error("got nil error for EXTINF with no comma, want error")
	}
}

func TestParseExtinfBadDuration(t *testing.T) {
	_, err := Parse(strings.NewReader("#EXTINF:not-a-number,Title\nsong.mp3\n"))
	if err == nil {
		t.Error("got nil error for non-numeric EXTINF duration, want error")
	}
}

func TestParseExtinfEmptyTitle(t *testing.T) {
	p, err := Parse(strings.NewReader("#EXTINF:10,\nsong.mp3\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Tracks[0].Title != "" || p.Tracks[0].Duration != 10*time.Second {
		t.Errorf("got %+v, want empty title with 10s duration", p.Tracks[0])
	}
}

func TestParseExtinfWithoutFollowingPath(t *testing.T) {
	// An EXTINF line at the very end of the file, with no path after it,
	// shouldn't produce a track: there's nothing to point at.
	p, err := Parse(strings.NewReader("song1.mp3\n#EXTINF:10,Dangling"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Tracks) != 1 {
		t.Fatalf("got %d tracks, want 1: %+v", len(p.Tracks), p.Tracks)
	}
}

func TestParseCommentsAndBlankLines(t *testing.T) {
	input := "#EXTM3U\n\n# a comment\nsong1.mp3\n\n"
	p, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Tracks) != 1 || p.Tracks[0].Path != "song1.mp3" {
		t.Fatalf("got %+v, want a single song1.mp3 track", p.Tracks)
	}
}

func TestParseErrorIncludesLineNumber(t *testing.T) {
	_, err := Parse(strings.NewReader("song1.mp3\n#EXTINF:bad,Title\nsong2.mp3\n"))
	if err == nil {
		t.Fatal("got nil error, want error")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("got error %q, want it to mention line 2", err.Error())
	}
}

func TestWriteRoundTrip(t *testing.T) {
	p := &Playlist{Tracks: []Track{
		{Path: "song1.mp3", Title: "Artist - Title", Duration: 123 * time.Second},
		{Path: "song2.mp3"},
	}}

	var buf bytes.Buffer
	if err := Write(&buf, p); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Parse(&buf)
	if err != nil {
		t.Fatalf("Parse of written output: %v", err)
	}
	if len(got.Tracks) != len(p.Tracks) {
		t.Fatalf("got %d tracks, want %d", len(got.Tracks), len(p.Tracks))
	}
	for i, tr := range p.Tracks {
		if got.Tracks[i] != tr {
			t.Errorf("track %d round-tripped as %+v, want %+v", i, got.Tracks[i], tr)
		}
	}
}
