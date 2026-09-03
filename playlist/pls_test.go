package playlist

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParsePLS(t *testing.T) {
	input := `[playlist]
File1=song1.mp3
Title1=Intro
Length1=42
File2=song2.mp3
Title2=Track Two
Length2=195
NumberOfEntries=2
Version=2
`
	p, err := ParsePLS(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParsePLS: %v", err)
	}
	want := []Track{
		{Path: "song1.mp3", Title: "Intro", Duration: 42 * time.Second},
		{Path: "song2.mp3", Title: "Track Two", Duration: 195 * time.Second},
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

func TestParsePLSOutOfOrder(t *testing.T) {
	// Nothing in the PLS format guarantees the FileN/TitleN/LengthN triples
	// appear in index order, or grouped together.
	input := `[playlist]
Title2=Track Two
File1=song1.mp3
Length1=42
File2=song2.mp3
Title1=Intro
Length2=195
`
	p, err := ParsePLS(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParsePLS: %v", err)
	}
	want := []string{"song1.mp3", "song2.mp3"}
	if len(p.Tracks) != len(want) {
		t.Fatalf("got %d tracks, want %d: %+v", len(p.Tracks), len(want), p.Tracks)
	}
	for i, path := range want {
		if p.Tracks[i].Path != path {
			t.Errorf("track %d: got path %q, want %q", i, p.Tracks[i].Path, path)
		}
	}
}

func TestParsePLSUnknownLength(t *testing.T) {
	input := `[playlist]
File1=stream.mp3
Title1=Radio
Length1=-1
`
	p, err := ParsePLS(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParsePLS: %v", err)
	}
	if len(p.Tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(p.Tracks))
	}
	if p.Tracks[0].Duration != 0 {
		t.Errorf("got duration %v, want 0 for unknown length", p.Tracks[0].Duration)
	}
}

func TestParsePLSTitleWithoutFile(t *testing.T) {
	// A TitleN or LengthN with no matching FileN isn't a track.
	input := `[playlist]
Title3=Orphan
File1=song1.mp3
`
	p, err := ParsePLS(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParsePLS: %v", err)
	}
	if len(p.Tracks) != 1 {
		t.Fatalf("got %d tracks, want 1: %+v", len(p.Tracks), p.Tracks)
	}
	if p.Tracks[0].Path != "song1.mp3" {
		t.Errorf("got path %q, want song1.mp3", p.Tracks[0].Path)
	}
}

func TestParsePLSBadLength(t *testing.T) {
	input := `[playlist]
File1=song1.mp3
Length1=not-a-number
`
	if _, err := ParsePLS(strings.NewReader(input)); err == nil {
		t.Error("got nil error for non-numeric Length, want error")
	}
}

func TestWritePLSRoundTrip(t *testing.T) {
	p := &Playlist{Tracks: []Track{
		{Path: "song1.mp3", Title: "Intro", Duration: 42 * time.Second},
		{Path: "song2.mp3"},
	}}

	var buf bytes.Buffer
	if err := WritePLS(&buf, p); err != nil {
		t.Fatalf("WritePLS: %v", err)
	}

	got, err := ParsePLS(&buf)
	if err != nil {
		t.Fatalf("ParsePLS of written output: %v", err)
	}
	if len(got.Tracks) != len(p.Tracks) {
		t.Fatalf("got %d tracks, want %d", len(got.Tracks), len(p.Tracks))
	}
	if got.Tracks[0].Path != "song1.mp3" || got.Tracks[0].Title != "Intro" || got.Tracks[0].Duration != 42*time.Second {
		t.Errorf("track 0 round-tripped as %+v", got.Tracks[0])
	}
	// Track 1 had no title, but WritePLS falls back to the path so PLS
	// readers that require TitleN always have something to show.
	if got.Tracks[1].Path != "song2.mp3" || got.Tracks[1].Title != "song2.mp3" {
		t.Errorf("track 1 round-tripped as %+v", got.Tracks[1])
	}
}
