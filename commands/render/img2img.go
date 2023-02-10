package render

import (
	"encoding/base64"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
	"github.com/diamondburned/arikawa/v3/discord"
)

var ErrChangingImg2ImgNotAllowed = errors.New("changing the Img2Img image is disabled")

func img2img(cmdctx *command.CommandContext, attachment discord.Attachment, data *sdapi.RenderData) error {
	res, err := http.Get(attachment.URL)
	if err != nil {
		return err
	}

	img, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	data.InitImage = "data:" + attachment.ContentType + ";base64," + base64.StdEncoding.EncodeToString(img)

	_, _ = cmdctx.TryReply("**Loaded Img2Img image from attachment!**")

	if attachment.Description != "" && config.CanChange(cmdctx.ChannelSettings.InUse, "promptstrength") == nil {
		f, err := strconv.ParseFloat(cmdctx.Args, 64)
		if err != nil {
			return err
		}

		cmdctx.ChannelSettings.PromptStrength = math.Min(math.Max(f, 0), 0.999_999)
	}

	data.PromptStrength = cmdctx.ChannelSettings.PromptStrength
	return nil
}
