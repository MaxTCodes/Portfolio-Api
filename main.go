package main

import (
	"backend/spotify"
	"backend/version"
	"backend/visuals"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	recover2 "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thanhpk/randstr"
	"log"
	"math/rand"
	"os"
	"time"

	// lol fewer lines = better right?
	_ "github.com/joho/godotenv/autoload"
)

type (
	info struct {
		Link, Name string
	}
	SongData struct {
		Artist, Song info
		Device       spotify.Device
		Playing      bool
	}
)

const (
	port             = ":5005"
	refreshTokenFile = ".refreshToken"
	refreshBaseTime  = 4
)

// all variables that get
var (
	refreshToken, authorizedIPHash string
	nowPlaying                     *spotify.NowPlaying
	updateNext                     time.Time
	client                         spotify.Client
)

var (
	clientID            = os.Getenv("CLIENT_ID")
	clientSecret        = os.Getenv("CLIENT_SECRET")
	adminPath           = randstr.String(15)
	updatePath          = randstr.String(5)
	adminCookieName     = randstr.String(10)
	adminCookiePassword = randstr.String(20)
)

func init() {
	rand.Seed(time.Now().UnixMicro())
}

func runBgTask() {
	if err := updateNowPlaying(); err != nil {
		log.Println("Failed to update Spotify Listening, Error: " + err.Error())
	}
}

func runPollingThread() {
	for {
		go runBgTask()
		// hard limit of 10 seconds, Spotify allows for approximately 180 requests per minute.
		<-time.After(time.Duration(refreshBaseTime+rand.Intn(6)) * time.Second)
	}
}

func main() {
	// fix that it shows up as errors in pm2
	log.SetOutput(os.Stdout)

	// make spotify client
	client = spotify.New(clientSecret, clientID)

	go runPollingThread()

	app := fiber.New(fiber.Config{
		AppName:               "Max's Portfolio Backend",
		RequestMethods:        []string{"GET", "POST", "HEAD", "OPTIONS"},
		DisableStartupMessage: true,
		// Because I use cloudflare as a proxy
		ProxyHeader: "CF-Connecting-IP",
		JSONDecoder: json.Unmarshal,
		JSONEncoder: json.Marshal,
	})

	//#region hooks
	app.Hooks().OnListen(func() error {
		visuals.StartUpMsg(version.GetVersionHash(), adminPath+"/"+updatePath)
		checkForAndLoadRefreshKey()
		return nil
	})
	//#endregion
	//#region middleware
	app.Use(recover2.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestCompression,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST",
		AllowHeaders: "*",
	}))
	//#endregion

	//#region normal user paths
	app.Get("/NowPlaying", func(c *fiber.Ctx) error {
		if nowPlaying == nil {
			return c.JSON(fiber.Map{
				"message": "Nothing is playing!",
				"success": false,
			})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"playingData": SongData{
				Artist: info{
					Link: nowPlaying.Item.Artists[0].ExternalUrls.Spotify,
					Name: nowPlaying.Item.Artists[0].Name,
				},
				Song: info{
					Link: nowPlaying.Item.ExternalUrls.Spotify,
					Name: nowPlaying.Item.Name,
				},
				Device:  nowPlaying.DeviceData,
				Playing: nowPlaying.IsPlaying,
			},
			"songEndTime": updateNext.Format(time.RFC3339),
		})
	})
	//#endregion

	//#region admin paths
	app.Get("/admin/spotify/callback/1", func(c *fiber.Ctx) error {
		if c.Cookies(adminCookieName, "") != adminCookiePassword || authorizedIPHash != getIPHash(c.IP()) {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		code := c.Query("code", "")
		if len(code) == 0 {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		token, err := client.GetRefreshToken(code, c.BaseURL()+app.GetRoute("callback1").Path)
		if err != nil {
			log.Printf("Error: %s", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
			})
		}
		refreshToken = token
		err = updateNowPlaying()
		if err != nil {
			log.Printf("Error: %s", err.Error())
			return c.Status(fiber.StatusFailedDependency).SendString("Failed to generate valid refresh token. Try again!")
		}
		log.Println("Successfully updated refresh token by request from " + c.IP())
		err = os.WriteFile(refreshTokenFile, []byte(token), 0644)
		if err != nil {
			log.Printf("Error: %s", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
			})
		}

		return c.SendString("Successfully Set Refresh Token")
	}).Name("callback1")

	admin := app.Group(adminPath)
	admin.Use(func(c *fiber.Ctx) error {
		// clear old cookies, so we don't use that much data :laugh:
		c.ClearCookie()
		c.Cookie(&fiber.Cookie{
			Name:    adminCookieName,
			Value:   adminCookiePassword,
			Expires: time.Now().Add(time.Hour * 24),
		})
		authorizedIPHash = getIPHash(c.IP())
		return c.Next()
	})
	admin.Get("/"+updatePath, func(c *fiber.Ctx) error {
		authURL := client.GetRefreshTokenURL(c.BaseURL() + app.GetRoute("callback1").Path)
		time.Sleep(time.Millisecond * 500)
		return c.Redirect(authURL, 301)
	})
	//#endregion

	// this one is a one-liner because it's so short
	if err := app.Listen(port); err != nil {
		// usually only happens when port is already being used
		panic(err)
	}

}

func getIPHash(ip string) string {
	hash := sha512.Sum512_256([]byte(ip))
	return hex.EncodeToString(hash[:])
}

func updateNowPlaying() error {
	np, err := client.GetNowPlaying(refreshToken)
	if err != nil || np == nil {
		if errors.Is(err, spotify.NotPlaying) {
			nowPlaying = nil
			return nil
		}
		return err
	}
	if np.IsPlaying {
		curTime := time.Unix(np.Timestamp/1000, 0)
		timeLeft := np.Item.DurationMs - np.ProgressMs
		updateNext = curTime.Add(time.Millisecond * time.Duration(timeLeft))
	}
	nowPlaying = np
	return nil
}

func checkForAndLoadRefreshKey() {
	if _, err := os.Stat(refreshTokenFile); err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		log.Println("No saved refresh token found!")
		return
	}
	f, err := os.ReadFile(refreshTokenFile)
	if err != nil {
		panic(err)
	}
	refreshToken = string(f)
	log.Println("Loaded refresh token from saved file")
}
