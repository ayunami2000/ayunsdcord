package commands

import (
	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var PromptCommand = command.NewCommand("prompt", []string{"p"}, promptCommandRun)

func promptCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current prompt:** %s", utils.StringOrNone(cmdctx.ChannelSettings.Prompt))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "prompt"); err != nil {
		return err
	}

	cmdctx.ChannelSettings.Prompt = utils.TruncateText(cmdctx.Args, 512)
	_, err := cmdctx.TryReply("**Prompt set to:** %s", cmdctx.ChannelSettings.Prompt)
	return err
}
