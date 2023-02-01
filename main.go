package main

import (
	"backend/spotify"
	"backend/version"
	"backend/visuals"
	"crypto/sha512"
	"encoding/hex"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	recover2 "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/thanhpk/randstr"
	"log"
	"os"
	"time"

	// lol fewer lines = better right?
	_ "github.com/joho/godotenv/autoload"
)

type (
	artist struct {
		Link, Name string
	}
	SongData struct {
		Artist  artist
		Name    string
		Playing bool
	}
)

const (
	port             = ":5005"
	refreshTokenFile = ".refreshToken"
)

var (
	refreshToken, authorizedIPHash string // both of these get defined later on in the app
	clientID                       = os.Getenv("CLIENT_ID")
	clientSecret                   = os.Getenv("CLIENT_SECRET")
	adminPath                      = randstr.String(15)
	updatePath                     = randstr.String(5)
	adminCookieName                = randstr.String(10)
	adminCookiePassword            = randstr.String(20)
	nowPlaying                     *spotify.NowPlaying
	updateNext                     time.Time
	lastUpdate                     time.Time
)

func init() {
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

func main() {
	// make spotify client
	client := spotify.New(clientSecret, clientID)

	app := fiber.New(fiber.Config{
		AppName:               "Max's Portfolio Backend",
		RequestMethods:        []string{"GET", "POST", "HEAD"},
		DisableStartupMessage: true,
		// Because I use cloudflare as a proxy
		ProxyHeader: "CF-Connecting-IP",
		JSONDecoder: json.Unmarshal,
		JSONEncoder: json.Marshal,
	})

	//#region hooks
	app.Hooks().OnListen(func() error {
		visuals.StartUpMsg(version.GetVersionHash(), adminPath+"/"+updatePath)
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

	// this has to be static because spotify requires a constant callback url
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
		err = updateNowPlaying(client)
		if err != nil {
			log.Printf("Error: %s", err.Error())
			return c.Status(fiber.StatusFailedDependency).JSON(fiber.Map{
				"success": false,
			})
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

	app.Get("/NowPlaying", func(c *fiber.Ctx) error {
		if updateNext.Before(time.Now()) || lastUpdate.Add(time.Second*15).Before(time.Now()) {
			err := updateNowPlaying(client)
			if err != nil {
				log.Printf("Error: %s", err.Error())
				return c.Status(fiber.StatusFailedDependency).JSON(fiber.Map{
					"success": false,
				})
			}
		}
		return c.JSON(fiber.Map{
			"success": true,
			"playingData": SongData{
				Artist: artist{
					Link: nowPlaying.Item.Artists[0].ExternalUrls.Spotify,
					Name: nowPlaying.Item.Artists[0].Name,
				},
				Name:    nowPlaying.Item.Name,
				Playing: nowPlaying.IsPlaying,
			},
			"songEndTime": updateNext.Format(time.RFC3339),
		})
	})

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

func updateNowPlaying(client spotify.Client) error {
	np, err := client.GetNowPlaying(refreshToken)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return err
	}
	if np.IsPlaying {
		curTime := time.Unix(np.Timestamp/1000, 0)
		timeLeft := np.Item.DurationMs - np.ProgressMs
		updateNext = curTime.Add(time.Millisecond * time.Duration(timeLeft))
	}
	nowPlaying = &np
	lastUpdate = time.Now()
	return nil
}
