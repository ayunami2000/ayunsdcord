package commands

import (
	"errors"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
)

var ModelCommand = command.NewCommand("model", []string{"m"}, modelCommandRun)
var ErrInvalidModel = errors.New("invalid model")

func modelCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current Model:** %s", cmdctx.ChannelSettings.Model)
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "model"); err != nil {
		return err
	}

	res, err := sdapi.GetModels()
	if err != nil {
		return err
	}

	model := ""
	for _, m := range res.Options.StableDiffusion {
		if strings.EqualFold(m, cmdctx.Args) {
			model = m
			break
		}
	}

	if model == "" {
		return ErrInvalidModel
	}

	cmdctx.ChannelSettings.Model = model

	_, err = cmdctx.TryReply("**Model set to:** %s", model)
	return err
}
