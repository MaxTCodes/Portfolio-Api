package main

import (
	"os"
	"time"

	"backend/shared"
	"backend/wrappers/spotify"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const SpotifyRefreshTime = 5

var (
	Logger       *zap.Logger
	refreshToken string
)

func updateNowPlaying() {
	np, err := shared.Spotify.GetNowPlaying(refreshToken)
	if err != nil {
		Logger.Error("Error getting now playing", zap.Error(err))
	}
	shared.NowPlaying.Set(np)
}

func pollSpotify() {
	for {
		go updateNowPlaying()
		<-time.After(time.Duration(SpotifyRefreshTime+(getRandomUint32()%6)) * time.Second)
	}
}

func main() {
	var err error
	// load dotenv
	if err = godotenv.Load(); err != nil {
		panic(err)
	}

	// load logger
	getLogger(os.Getenv("ENVIRONMENT") == "development")

	// load spotify
	shared.Spotify = spotify.New(os.Getenv("SPOTIFY_SECRET"), os.Getenv("SPOTIFY_ID"))

	go pollSpotify()

	defer func(Logger *zap.Logger) {
		err := Logger.Sync()
		if err != nil {
			panic(err)
		}
	}(Logger)

	// set variable port
	port := os.Getenv("PORT")

	app := fiber.New(fiber.Config{
		Prefork:     false,
		AppName:     "Portfolio V2.0.0",
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	if err := app.Listen(":" + port); err != nil {
		Logger.Fatal("Error listening", zap.Error(err))
	}
}

func getLogger(dev bool) {
	var lf func(option ...zap.Option) (*zap.Logger, error)
	if dev {
		lf = zap.NewDevelopment
	} else {
		lf = zap.NewProduction
	}
	Logger = zap.Must(lf())
}

func getRandomUint32() uint32 {
	x := time.Now().UnixNano()
	return uint32((x >> 32) ^ x)
}
