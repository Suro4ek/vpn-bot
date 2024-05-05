package main

import (
	"flag"
	"log"
	"strconv"
	"strings"
	"time"
	"vpn-bot/internal/commands"
	"vpn-bot/pkg/wireguard"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v3"
	"gopkg.in/telebot.v3/middleware"
)

func main() {
	var envPath string
	flag.StringVar(&envPath, "envfile", ".env", "envFile path")
	flag.Parse()
	wg := wireguard.NewParserConfig()
	cfg, err := wg.LoadConfig("./wg0.json")
	if err != nil {
		panic(err)
	}
	envFile, err := godotenv.Read(envPath)
	if err != nil {
		panic(err)
	}

	pref := tele.Settings{
		Token:  envFile["TOKEN"],
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	wgConfig := wireguard.NewWireguardConfig(cfg)
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	b.Use(middleware.AutoRespond())
	if envFile["GROUP"] == "" {
		tgUsers := make([]int64, 0)
		for _, tgUserEnv := range strings.Split(envFile["TG_USERS"], ",") {
			tgUser, err := strconv.Atoi(tgUserEnv)
			if err != nil {
				continue
			}
			tgUsers = append(tgUsers, int64(tgUser))
		}
		b.Use(middleware.Whitelist(tgUsers...))
	} else {
		tgChannel, err := strconv.Atoi(envFile["GROUP"])
		if err != nil {
			return
		}
		b.Use(WhitelistChats(int64(tgChannel)))
	}

	ch := commands.NewCommandHandler(wgConfig)
	ch.Register(b)

	b.Start()
}

func Restrict(v middleware.RestrictConfig) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		if v.In == nil {
			v.In = next
		}
		if v.Out == nil {
			v.Out = next
		}
		return func(c tele.Context) error {
			for _, chat := range v.Chats {
				if chat == c.Chat().ID {
					return v.In(c)
				}
			}
			return v.Out(c)
		}
	}
}


func WhitelistChats(chats ...int64) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return Restrict(middleware.RestrictConfig{
			Chats: chats,
			In:    next,
			Out:   func(c tele.Context) error { return nil },
		})(next)
	}
}
