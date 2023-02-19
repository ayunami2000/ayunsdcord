package commands

import (
	"errors"
	"log"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/kobold"
)

var KoboldCommand = command.NewCommand("kobold", []string{"kb"}, koboldRun)
var ErrKoboldDisabled = errors.New("kobold is disabled")

func koboldRun(cmdctx *command.CommandContext) error {
	if !config.Config.KoboldEnabled {
		return ErrKoboldDisabled
	}

	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Please specify a prompt!**")
		return err
	}

	data := &kobold.KoboldRequest{
		Prompt:      cmdctx.Args,
		Temperature: 0.7,
		TopP:        1.0,
	}

	res, err := kobold.Generate(data)
	if err != nil {
		log.Println("Could not query kobold:", err)
		return err
	}

	_, err = cmdctx.TryReply("**Kobold:** %s", res)
	return err
}
