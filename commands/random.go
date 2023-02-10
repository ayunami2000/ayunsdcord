package commands

import (
	"strconv"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/commands/render"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
	"github.com/tjarratt/babble"
)

var RandomCommand = command.NewCommand("random", []string{"rand", "randomrender", "rr"}, randomCommandRun)

var babbler = babble.NewBabbler()

func init() {
	babbler.Separator = ", "
}

func randomCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		babbler.Count = 10
	} else {
		i, err := strconv.Atoi(cmdctx.Args)
		if err != nil {
			return err
		} else {
			if i < 1 {
				i = 1
			} else if i > 100 {
				i = 100
			}
			babbler.Count = i
		}
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "prompt"); err != nil {
		return err
	}

	cmdctx.ChannelSettings.Prompt = utils.TruncateText(babbler.Babble(), 512)
	_, err := cmdctx.TryReply("**Prompt randomly set to:** %s", cmdctx.ChannelSettings.Prompt)
	if err != nil {
		return err
	}

	if cmdctx.CalledWithAlias == "randomrender" || cmdctx.CalledWithAlias == "rr" {
		return render.Run(cmdctx)
	}

	return nil
}
