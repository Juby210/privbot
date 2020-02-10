package main

import (
	"fmt"

	"github.com/andersfylling/disgord"
)

func starboard(s disgord.Session, r *disgord.MessageReactionAdd) {
	if r.PartialEmoji.Name != "⭐" || r.ChannelID == config.Starboard {
		return
	}

	msg, err := s.GetMessage(ctx, r.ChannelID, r.MessageID)
	if err != nil {
		return
	}

	if msg.Content == "" && len(msg.Attachments) == 0 {
		return
	}

	var total uint
	for _, rr := range msg.Reactions {
		if rr.Emoji.Name == "⭐" {
			total = rr.Count
		}
	}

	messages, err := s.GetMessages(ctx, config.Starboard, &disgord.GetMessagesParams{})
	if err != nil {
		return
	}

	a, _ := msg.Author.AvatarURL(128, false)
	c, _ := s.GetChannel(ctx, r.ChannelID)
	e := &disgord.Embed{
		Author: &disgord.EmbedAuthor{
			Name: msg.Author.Tag(), IconURL: a,
		},
		Description: fmt.Sprintf("%s\n\n[[link]](https://discordapp.com/channels/%d/%d/%d)",
			msg.Content, c.GuildID, c.ID, msg.ID),
		Footer: &disgord.EmbedFooter{
			Text: msg.ID.String(),
		},
		Timestamp: msg.Timestamp,
		Color:     16777130,
	}

	if len(msg.Attachments) > 0 && msg.Attachments[0].Width != 0 {
		e.Image = &disgord.EmbedImage{URL: msg.Attachments[0].URL}
	} else if len(msg.Embeds) > 0 && msg.Embeds[0].Type == "image" {
		e.Image = &disgord.EmbedImage{URL: msg.Embeds[0].URL}
	}

	star := "⭐"
	if total >= 10 {
		star = "✨"
	} else if total >= 5 {
		star = "💫"
	} else if total >= 3 {
		star = "🌟"
	}

	u, _ := s.GetCurrentUser(ctx)
	content := fmt.Sprintf("%s %d | <#%d>", star, total, c.ID)
	for _, m := range messages {
		if m.Author.ID == u.ID &&
			len(m.Embeds) == 1 &&
			m.Embeds[0].Footer.Text == msg.ID.String() {
			if m.Content != content {
				s.UpdateMessage(ctx, m.ChannelID, m.ID).SetContent(content).Execute()
			}
			return
		}
	}
	s.SendMsg(ctx, config.Starboard, disgord.CreateMessageParams{Content: content, Embed: e})
}
