package commands

import (
	"errors"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var VALID_UPSCALERS = []string{"RealESRGAN_x4plus", "RealESRGAN_x4plus_anime_6B"}
var UpscalerCommand = command.NewCommand("upscaler", []string{"u"}, upscalerRun)
var ErrInvalidUpscaler = errors.New("invalid upscaler")

func upscalerRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply(`**Current upscaler:** %s
Upscalers: %s`, utils.StringOrNone(cmdctx.ChannelSettings.Upscaler), strings.Join(VALID_UPSCALERS, ", "))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "upscaler"); err != nil {
		return err
	}

	upscaler := ""
	for _, u := range VALID_UPSCALERS {
		if strings.EqualFold(u, cmdctx.Args) {
			upscaler = u
			break
		}
	}

	if upscaler == "" {
		return ErrInvalidUpscaler
	}

	cmdctx.ChannelSettings.Upscaler = upscaler

	_, err := cmdctx.TryReply("**Upscaler set to:** %s", upscaler)
	return err
}
