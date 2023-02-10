package commands

import (
	"errors"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var VaeCommand = command.NewCommand("vae", []string{"v"}, vaeCommandRun)
var ErrInvalidVae = errors.New("invalid vae")

func vaeCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current VAE:** %s", utils.StringOrNone(cmdctx.ChannelSettings.VAE))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "vae"); err != nil {
		return err
	}

	res, err := sdapi.GetModels()
	if err != nil {
		return err
	}

	vae := ""
	for _, v := range res.Options.VAE {
		if strings.EqualFold(v, cmdctx.Args) {
			vae = v
			break
		}
	}

	if vae == "" {
		return ErrInvalidVae
	}

	cmdctx.ChannelSettings.VAE = vae

	_, err = cmdctx.TryReply("**VAE set to:** %s", vae)
	return err
}
