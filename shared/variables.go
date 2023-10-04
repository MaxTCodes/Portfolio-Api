package shared

import "backend/wrappers/spotify"

var (
	NowPlaying spotify.NowPlayingSafe
	Spotify    *spotify.Client
)
