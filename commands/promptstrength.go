package commands

import (
	"math"
	"strconv"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
)

var PromptStrengthCommand = command.NewCommand("promptstrength", []string{"ps"}, promptStrengthRun)

func promptStrengthRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current Img2Img prompt strength:** %g", cmdctx.ChannelSettings.PromptStrength)
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "promptstrength"); err != nil {
		return err
	}

	f, err := strconv.ParseFloat(cmdctx.Args, 64)
	if err != nil {
		return err
	}

	f = math.Min(math.Max(f, 0), 0.999_999)

	cmdctx.ChannelSettings.PromptStrength = f
	_, err = cmdctx.TryReply("**Img2Img prompt strength set to:** %g", f)
	return err
}
