package telegram

import (
	"context"
	"log/slog"
	"time"
)

type MessageHandler interface {
	Handle(context.Context, int64, string) string
}

type Bot struct {
	client  *Client
	handler MessageHandler
	logger  *slog.Logger
}

func NewBot(client *Client, handler MessageHandler, logger *slog.Logger) *Bot {
	return &Bot{client: client, handler: handler, logger: logger}
}

func (bot *Bot) Run(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		updates, err := bot.client.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			bot.logger.Error("telegram polling failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message == nil || update.Message.Text == "" {
				continue
			}
			reply := bot.handler.Handle(ctx, update.Message.From.ID, update.Message.Text)
			if err := bot.client.SendMessage(ctx, update.Message.Chat.ID, reply); err != nil {
				bot.logger.Error("telegram reply failed", "error", err, "chat_id", update.Message.Chat.ID)
			}
		}
	}
}
