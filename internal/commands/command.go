package commands

import (
	"fmt"
	"vpn-bot/pkg/wireguard"

	tele "gopkg.in/telebot.v3"
)

type Context interface {
	SendMessage(message string)
}

type Command interface {
	Execute(c Context)
}

type CommandHandler struct{
	wg wireguard.WiregaurdConfig

}

func NewCommandHandler(wg wireguard.WiregaurdConfig) CommandHandler{
	return CommandHandler{
		wg: wg,
	}
}

func (ch CommandHandler) Register(b *tele.Bot) {
	b.Handle("/start", func(c tele.Context) error {
		defer c.Respond()
		return c.Send("Я бот для впн\n и у меня есть команды\n /add [имя] - добавить клиента \n /remove [имя] - удалить клиента \n /list - список клиентов")
	})
	b.Handle("/add", func(c tele.Context) error {
		defer c.Respond()
		clientName := c.Message().Payload
		err := ch.wg.AddClient(clientName)
		if err != nil {
			fmt.Println(err)
			return c.Send("Не удалось добавить клиента")
		}
		a := &tele.Document{
			File: tele.FromDisk(fmt.Sprintf("client-%s.conf", clientName,)),
			FileName: fmt.Sprintf("vpn-%s.conf", clientName,),
		}
		fmt.Println(a)
		return c.Send(a)
	})
	b.Handle("/remove", func(c tele.Context) error {
		defer c.Respond()
		clientName := c.Message().Payload
		err := ch.wg.RemoveClient(clientName)
		if err != nil {
			fmt.Println(err)
			return c.Send("Не удалось удалить клиента")
		}
		return c.Send("Клиент успешно удален")
	})
	b.Handle("/list", func(c tele.Context) error {
		defer c.Respond()
		clients := ch.wg.GetClients()
		message := "Список клиентов: \n"
		for _, client := range clients {
			message += "- "+client + " \n"
		}
		return c.Send(message)
	})
}