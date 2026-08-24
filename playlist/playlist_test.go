package playlist

import (
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
