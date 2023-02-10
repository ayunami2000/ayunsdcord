package render

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

var ErrFailedToGetAttachmentURL = errors.New("failed to get attachment URL")

func frameEmbed(cmdctx *command.CommandContext, oldMessage *discord.Message, url string, step uint, totalSteps uint, hasInitImage bool) error {
	settings := cmdctx.ChannelSettings

	footer := fmt.Sprintf("Step %d of %d", step, totalSteps)
	if step >= totalSteps {
		footer = "Done!"
	}

	if url != "" {
		settings.CurrentRenderInfo.LastFrameUrl = url
	}

	desc := fmt.Sprintf("**Prompt:** %s", settings.Prompt)
	if settings.NegativePrompt != "" {
		desc += fmt.Sprintf("\n**Negative Prompt:** %s", settings.NegativePrompt)
	}

	desc += fmt.Sprintf(`
**Width:** %d
**Height**: %d
**Inference Steps:** %d
**Guidance Scale:** %g
**Model:** %s`, settings.Width, settings.Height, settings.InferenceSteps, settings.GuidanceScale, settings.Model)

	if settings.VAE != "" {
		desc += fmt.Sprintf("\n**VAE:** %s", settings.VAE)
	}
	if settings.HyperNetwork != "" {
		desc += fmt.Sprintf("\n**HyperNetwork:** %s", settings.HyperNetwork)
	}
	if settings.Upscaler != "" {
		desc += fmt.Sprintf("\n**Upscaler:** %dx %s", settings.UpscaleAmount, settings.Upscaler)
	}

	if hasInitImage {
		desc += fmt.Sprintf("\n**Img2Img Prompt Strength:** %g", settings.PromptStrength)
	}

	_, err := cmdctx.Executor.State.EditMessageComplex(oldMessage.ChannelID, oldMessage.ID, api.EditMessageData{
		Content: option.NewNullableString(""),
		Embeds: &[]discord.Embed{{
			Title:       "Stable Diffusion",
			Description: desc,
			Footer: &discord.EmbedFooter{
				Text: footer,
			},
			Image: &discord.EmbedImage{
				URL: settings.CurrentRenderInfo.LastFrameUrl,
			},
			Timestamp: discord.NewTimestamp(time.Now()),
		}},
	})

	return err
}

func frame(cmdctx *command.CommandContext, oldMessage *discord.Message, reader io.Reader, step uint, totalSteps uint, hasInitImage bool) (*discord.Message, error) {
	if reader == nil {
		return nil, frameEmbed(cmdctx, oldMessage, "", step, totalSteps, hasInitImage)
	}

	ext := "jpg"
	if step == totalSteps {
		ext = "png"
	}

	if config.Config.FrameUrl != "" {
		body, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		cmdctx.ChannelSettings.CurrentRenderInfo.FrameData = body
		return nil, frameEmbed(cmdctx, oldMessage, fmt.Sprintf("%s/%s/%d.%s", config.Config.FrameUrl, oldMessage.ChannelID, time.Now().UnixNano(), ext), step, totalSteps, hasInitImage)
	}

	dumpChannel := config.GetImageDumpChannelId()
	if dumpChannel == discord.NullChannelID {
		dumpChannel = oldMessage.ChannelID
	}

	msg, err := cmdctx.Executor.SendMessageComplex(dumpChannel, api.SendMessageData{
		Files: []sendpart.File{{
			Name:   fmt.Sprintf("stable-diffusion_%d.%s", time.Now().UnixNano(), ext),
			Reader: reader,
		}},
	})

	if err != nil {
		return nil, err
	} else if len(msg.Attachments) < 1 {
		return nil, ErrFailedToGetAttachmentURL
	}

	return msg, frameEmbed(cmdctx, oldMessage, msg.Attachments[0].URL, step, totalSteps, hasInitImage)
}
