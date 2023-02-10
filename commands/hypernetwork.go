package commands

import (
	"errors"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var HyperNetworkCommand = command.NewCommand("hypernetwork", []string{"hn"}, hyperNetworkRun)
var ErrInvalidHyperNetwork = errors.New("invalid HyperNetwork")

func hyperNetworkRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current HyperNetwork:** %s", utils.StringOrNone(cmdctx.ChannelSettings.HyperNetwork))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "hypernetwork"); err != nil {
		return err
	}

	res, err := sdapi.GetModels()
	if err != nil {
		return err
	}

	hypernetwork := ""
	for _, h := range res.Options.HyperNetwork {
		if strings.EqualFold(h, cmdctx.Args) {
			hypernetwork = h
			break
		}
	}

	if hypernetwork == "" {
		return ErrInvalidHyperNetwork
	}

	cmdctx.ChannelSettings.HyperNetwork = hypernetwork

	_, err = cmdctx.TryReply("**HyperNetwork set to:** %s", hypernetwork)
	return err
}
