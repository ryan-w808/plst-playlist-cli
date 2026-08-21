# plst

A small Go library and CLI for reading M3U/M3U8 audio playlists.

Most tools that touch playlists assume the file is sitting on disk, which
is annoying the moment you want to pipe one in from `curl`, a zip extractor,
or another program in a shell pipeline. `plst` reads from a named file or
from stdin using the same code path, so either works without special
casing on the caller's side.

## Install

```
go install plst/cmd/plst
```

Or just clone the repo and `go build ./cmd/plst`.

## Usage

From a file:

```
$ plst my-mix.m3u
  1. Intro (0:42)
  2. Track Two (3:15)
  3. Track Three (4:01)
```

From stdin, explicit `-`:

```
$ curl -s https://example.com/radio.m3u8 | plst -
```

From stdin, no argument at all (same behavior as `-`):

```
$ cat playlists/*.m3u | plst
```

## Playlist format

`plst` understands the two M3U conventions in the wild:

Plain, one path per line:

```
song1.mp3
song2.mp3
```

Extended, with a duration and title before each path:

```
#EXTM3U
#EXTINF:42,Intro
song1.mp3
#EXTINF:195,Track Two
song2.mp3
```

The two styles can be mixed in the same file. Lines starting with `#` that
aren't `#EXTINF` (like the `#EXTM3U` header) are skipped.

## Library

The parser is in `playlist` and works on anything that implements
`io.Reader`, which is what lets the CLI treat files and stdin identically:

```go
list, err := playlist.Parse(r)
if err != nil {
	// handle error
}
for _, t := range list.Tracks {
	fmt.Println(t.Path, t.Title, t.Duration)
}
```

## Status

Early. Parsing works; writing playlists back out and format conversion
don't exist yet.

## License

MIT, see LICENSE.
