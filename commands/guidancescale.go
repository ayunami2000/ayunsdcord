package commands

import (
	"math"
	"strconv"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
)

var GuidanceScaleCommand = command.NewCommand("guidancescale", []string{"gs"}, guidanceScaleRun)

func guidanceScaleRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current guidance scale:** %g", cmdctx.ChannelSettings.GuidanceScale)
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "guidancescale"); err != nil {
		return err
	}

	f, err := strconv.ParseFloat(cmdctx.Args, 64)
	if err != nil {
		return err
	}

	f = math.Min(math.Max(f, 1.1), 50)

	cmdctx.ChannelSettings.GuidanceScale = f
	_, err = cmdctx.TryReply("**Guidance scale set to:** %g", f)
	return err
}
