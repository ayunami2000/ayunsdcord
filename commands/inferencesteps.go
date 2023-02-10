package commands

import (
	"math"
	"strconv"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
)

var InferenceStepsCommand = command.NewCommand("inferencesteps", []string{"is"}, inferenceStepsRun)

func inferenceStepsRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Current inference steps:** %d", cmdctx.ChannelSettings.InferenceSteps)
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "inferencesteps"); err != nil {
		return err
	}

	i, err := strconv.ParseUint(cmdctx.Args, 10, 64)
	if err != nil {
		return err
	}

	cmdctx.ChannelSettings.InferenceSteps = uint(math.Min(math.Max(float64(i), 1), 100))
	_, err = cmdctx.TryReply("**Inference steps set to:** %d", cmdctx.ChannelSettings.InferenceSteps)
	return err
}
