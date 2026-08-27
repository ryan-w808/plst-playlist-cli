package playlist

import (
	"math/rand"
	"path/filepath"
	"testing"
)

func TestDedupe(t *testing.T) {
	p := &Playlist{
		Tracks: []Track{
			{Path: "song1.mp3"},
			{Path: "song2.mp3"},
			{Path: "song1.mp3"},
			{Path: "a/./song3.mp3"},
			{Path: "a/song3.mp3"},
			{Path: "song2.mp3", Title: "different metadata, same path"},
		},
	}

	p.Dedupe()

	want := []string{"song1.mp3", "song2.mp3", "a/song3.mp3"}
	if len(p.Tracks) != len(want) {
		t.Fatalf("got %d tracks, want %d: %+v", len(p.Tracks), len(want), p.Tracks)
	}
	for i, path := range want {
		if p.Tracks[i].Path != path {
			t.Errorf("track %d: got path %q, want %q", i, p.Tracks[i].Path, path)
		}
	}
	// The first occurrence's metadata is the one kept.
	if p.Tracks[1].Title != "" {
		t.Errorf("track 1: got title %q, want empty (first occurrence had no title)", p.Tracks[1].Title)
	}
}

func TestDedupeEmpty(t *testing.T) {
	p := &Playlist{}
	p.Dedupe()
	if len(p.Tracks) != 0 {
		t.Errorf("got %d tracks, want 0", len(p.Tracks))
	}
}

func TestShuffleIsPermutation(t *testing.T) {
	p := &Playlist{Tracks: []Track{
		{Path: "a"}, {Path: "b"}, {Path: "c"}, {Path: "d"}, {Path: "e"},
	}}
	orig := append([]Track(nil), p.Tracks...)

	p.Shuffle(rand.New(rand.NewSource(1)))

	if len(p.Tracks) != len(orig) {
		t.Fatalf("got %d tracks after shuffle, want %d", len(p.Tracks), len(orig))
	}
	seen := make(map[string]bool, len(p.Tracks))
	for _, tr := range p.Tracks {
		seen[tr.Path] = true
	}
	for _, tr := range orig {
		if !seen[tr.Path] {
			t.Errorf("track %q missing after shuffle", tr.Path)
		}
	}
}

func TestShuffleDeterministic(t *testing.T) {
	newList := func() *Playlist {
		return &Playlist{Tracks: []Track{
			{Path: "a"}, {Path: "b"}, {Path: "c"}, {Path: "d"},
			{Path: "e"}, {Path: "f"}, {Path: "g"}, {Path: "h"},
		}}
	}

	p1 := newList()
	p1.Shuffle(rand.New(rand.NewSource(42)))

	p2 := newList()
	p2.Shuffle(rand.New(rand.NewSource(42)))

	for i := range p1.Tracks {
		if p1.Tracks[i].Path != p2.Tracks[i].Path {
			t.Fatalf("same seed produced different order at index %d: %q vs %q", i, p1.Tracks[i].Path, p2.Tracks[i].Path)
		}
	}
}

func TestShuffleEmpty(t *testing.T) {
	p := &Playlist{}
	p.Shuffle(rand.New(rand.NewSource(1)))
	if len(p.Tracks) != 0 {
		t.Errorf("got %d tracks, want 0", len(p.Tracks))
	}
}

func TestResolve(t *testing.T) {
	p := &Playlist{
		Tracks: []Track{
			{Path: "song1.mp3"},
			{Path: "../shared/song2.mp3"},
			{Path: filepath.FromSlash("/absolute/song3.mp3")},
			{Path: "http://example.com/stream.mp3"},
			{Path: ""},
		},
	}

	p.Resolve(filepath.FromSlash("/music/rock"))

	want := []string{
		filepath.FromSlash("/music/rock/song1.mp3"),
		filepath.FromSlash("/music/shared/song2.mp3"),
		filepath.FromSlash("/absolute/song3.mp3"),
		"http://example.com/stream.mp3",
		"",
	}
	for i, path := range want {
		if p.Tracks[i].Path != path {
			t.Errorf("track %d: got path %q, want %q", i, p.Tracks[i].Path, path)
		}
	}
}
