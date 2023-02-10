package commands

import (
	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var NegativePromptCommand = command.NewCommand("negativeprompt", []string{"np"}, negativePromptCommandRun)

func negativePromptCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current negative prompt:** %s", utils.StringOrNone(cmdctx.ChannelSettings.NegativePrompt))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "negativeprompt"); err != nil {
		return err
	}

	cmdctx.ChannelSettings.NegativePrompt = utils.TruncateText(cmdctx.Args, 512)
	_, err := cmdctx.TryReply("**Negative prompt set to:** %s", cmdctx.ChannelSettings.NegativePrompt)
	return err
}
