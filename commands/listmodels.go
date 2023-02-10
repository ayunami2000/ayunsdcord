package commands

import (
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/sdapi"
)

var ListModelsCommand = command.NewCommand("listmodels", []string{"lm"}, listModelsCommandRun)

func listModelsCommandRun(cmdctx *command.CommandContext) error {
	res, err := sdapi.GetModels()
	if err != nil {
		return err
	}

	_, err = cmdctx.TryReply(`**Models:**
__Stable Diffusion__: %s
__VAE__: %s
__HyperNetwork__: %s`,
		strings.Join(res.Options.StableDiffusion, ", "),
		strings.Join(res.Options.VAE, ", "),
		strings.Join(res.Options.HyperNetwork, ", "))

	return err
}
