package main

import (
	"fmt"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/bot"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/utils/json/option"
)

func starboard(ctx *bot.Context) func(r *gateway.MessageReactionAddEvent) {
	return func(r *gateway.MessageReactionAddEvent) {
		if r.Emoji.Name != "⭐" || r.ChannelID == config.Starboard {
			return
		}

		msg, err := ctx.Message(r.ChannelID, r.MessageID)
		if err != nil {
			return
		}

		if msg.Content == "" && len(msg.Attachments) == 0 {
			return
		}

		total := 0
		for _, rr := range msg.Reactions {
			if rr.Emoji.Name == "⭐" {
				total = rr.Count
			}
		}

		messages, err := ctx.Messages(config.Starboard)
		if err != nil {
			return
		}

		star := "⭐"
		if total >= 10 {
			star = "✨"
		} else if total >= 5 {
			star = "💫"
		} else if total >= 3 {
			star = "🌟"
		}

		content := fmt.Sprintf("%s %d | <#%s>", star, total, r.ChannelID)
		for _, m := range messages {
			if m.Author.ID == ctx.Ready.User.ID &&
				len(m.Embeds) == 1 &&
				m.Embeds[0].Footer.Text == msg.ID.String() {
				if m.Content != content {
					ctx.EditMessageComplex(config.Starboard, m.ID,
						api.EditMessageData{Content: option.NewNullableString(content)})
				}
				return
			}
		}

		e := &discord.Embed{
			Author: &discord.EmbedAuthor{
				Name: fmt.Sprintf("%s#%s", msg.Author.Username, msg.Author.Discriminator),
				Icon: msg.Author.AvatarURL(),
			},
			Description: fmt.Sprintf("%s\n\n[[link]](%s)", msg.Content, msg.URL()),
			Footer: &discord.EmbedFooter{
				Text: msg.ID.String(),
			},
			Timestamp: msg.Timestamp,
			Color:     16777130,
		}

		if len(msg.Attachments) > 0 && msg.Attachments[0].Width != 0 {
			e.Image = &discord.EmbedImage{URL: msg.Attachments[0].URL}
		} else if len(msg.Embeds) > 0 && msg.Embeds[0].Type == discord.ImageEmbed {
			e.Image = &discord.EmbedImage{URL: msg.Embeds[0].URL}
		}

		ctx.SendMessage(config.Starboard, content, e)
	}
}
