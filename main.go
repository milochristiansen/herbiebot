/*
Copyright 2018 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

// Herbie: Heretical Edge new post Discord notification bot.
package main

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/bwmarrin/discordgo"

	"github.com/glebarez/sqlite"
	"github.com/milochristiansen/sessionlogger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// https://discordapp.com/oauth2/authorize?client_id=402521174384574464&scope=bot&permissions=133120
var (
	APIKey string
	Site   = "https://ceruleanscrawling.wordpress.com"
	Feeds  = []Feed{
		{"/category/summus-proelium/feed", "543593314746761228", "<@&850455939625517096>"},
		{"/category/uncategorized/feed", "383419886250098691", "@everyone"},
		{"/category/heretical-edge/feed", "383419886250098691", "<@&850455420912140320>"},
		//{"/feed", []string{"383419886250098691"}, "@everyone"}, // Site wide feed. No longer used.
	}
)

type Feed struct {
	URL      string
	Channel string
	Role     string
}

var Log *sessionlogger.Logger

type Entry struct {
	ID uint

	URL  string
	Channel string
	Role string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Spin up the server.

	Log = sessionlogger.NewMasterLogger()
	Log.I.Println("Starting Herbie!")

	// GORM makes tracking what we have seen already easy.
	Log.I.Println("Connecting GORM database...")
	db, err := gorm.Open(sqlite.Open("herbie.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		Log.E.Println("Error opening GORM DB:", err)
		os.Exit(1)
	}
	db.AutoMigrate(&Entry{})

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + APIKey)
	if err != nil {
		Log.E.Println("Error creating Discord session:", err)
		return
	}
	dg.Identify.Intents |= discordgo.IntentMessageContent

	dg.AddHandler(messageCreate)
	dg.AddHandler(onConnect)

	Log.I.Println("Connecting to Discord...")
	err = dg.Open()
	if err != nil {
		Log.E.Println("Error opening Discord connection:", err)
		return
	}

	Log.I.Println("Initialization finished, entering long run phase.")

	fp := gofeed.NewParser()
	for {
		// Get a list of all new articles in all feeds.
		notify := []*Entry{}
		foundanything := false
		for _, fdata := range Feeds {
			feed, err := fp.ParseURL(Site + fdata.URL)
			if err != nil {
				Log.W.Println("Error reading RSS feed:", err)
				break
			}

			for _, item := range feed.Items {
				entry := &Entry{0, item.Link, fdata.Channel, fdata.Role}

				dbentry := &Entry{}
				err := db.First(&dbentry, "URL = ?", entry.URL).Error

				// If the item isn't in the DB
				if errors.Is(err, gorm.ErrRecordNotFound) {
					notify = append(notify, entry)
					// Add to DB later.
					continue
				}

				// If we had some other error
				if err != nil {
					Log.W.Println(err)
					goto end
				}

				// The item is already in the DB
				foundanything = true
			}
		}

		if len(notify) > 0 {
			Log.I.Printf("Sending %v entries.\n", len(notify))

			for _, notification := range notify {
				if foundanything {
					_, err := dg.ChannelMessageSend(notification.Channel, notification.Role+" New Post: "+notification.URL)
					if err != nil {
						Log.W.Println("Error sending message to:", notification.Channel, err)
					}
				}

				// We sent the message, add to DB.
				err = db.Create(&notification).Error
				if err != nil {
					Log.W.Println(err)
					continue
				}
			}

			if !foundanything {
				// Do not notify. If it looks like the entire site inventory is new, it probably
				// just means that the DB is gone and we are generating a new one.
				Log.W.Println("No listings found in the DB, no notifications. Was the DB missing?")
				// Just fall through
			}
		}

		// Filter out all known articles.

	end:
		time.Sleep(1 * time.Minute)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch m.Content {
	case "Hey Herbie!":
		linelist, err := ioutil.ReadFile("herbie.quotes")
		if err != nil {
			return
		}
		lines := strings.Split(string(linelist), "\n")
		nlines := []string{}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			nlines = append(nlines, line)
		}
		if len(nlines) > 0 {
			_, err := s.ChannelMessageSend(m.ChannelID, nlines[rand.Intn(len(nlines))])
			if err != nil {
				Log.W.Println("Error responding to hey from:", m.ChannelID, err)
			}
		}
	case "Herbie?":
		_, err := s.ChannelMessageSend(m.ChannelID, "Try: `Hey Herbie!`. Herbie may also do fun things if you wish him a happy birthday at the right time of year...")
		if err != nil {
			Log.W.Println("Error responding to question from:", m.ChannelID, err)
		}
	default:
		t, msg := time.Now(), strings.ToLower(m.Content)
		// September 4th, the day Flick throws Herbie through the portal.
		if t.Month() == time.September && t.Day() == 4 {
			if strings.Contains(msg, "happy") && strings.Contains(msg, "birthday") && strings.Contains(msg, "herbie") {
				_, err := s.ChannelMessageSend(m.ChannelID, "Herbie seems pleased with your greeting.")
				if err != nil {
					Log.W.Println("Error responding to birthday wish from:", m.ChannelID, err)
				}
			}
		}
	}
}

func onConnect(s *discordgo.Session, r *discordgo.Ready) {
	err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{{
			Name: "Type Herbie? for help.",
			Type: discordgo.ActivityTypeCustom,
		}},
	})
	if err != nil {
		Log.W.Println("Error setting status:", err)
	}
}
