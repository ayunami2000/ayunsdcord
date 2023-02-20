package commands

import (
	"errors"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
)

var VALID_SAMPLERS = []string{"plms", "ddim", "heun", "euler", "euler_a", "dpm2", "dpm2_a", "lms", "dpm_solver_stability", "dpmpp_2s_a", "dpmpp_2m", "dpmpp_sde", "dpm_fast", "dpm_adaptive", "unipc_snr", "unipc_tu", "unipc_snr_2", "unipc_tu_2", "unipc_tq"}
var SamplerCommand = command.NewCommand("sampler", []string{"sm"}, samplerCommandRun)
var ErrInvalidSampler = errors.New("invalid sampler")

func samplerCommandRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply(`**Current Sampler:** %s
Samplers: %s`, cmdctx.ChannelSettings.Sampler, strings.Join(VALID_SAMPLERS, ", "))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "sampler"); err != nil {
		return err
	}

	sampler := ""
	for _, s := range VALID_SAMPLERS {
		if strings.EqualFold(s, cmdctx.Args) {
			sampler = s
			break
		}
	}

	if sampler == "" {
		return ErrInvalidSampler
	}

	cmdctx.ChannelSettings.Sampler = sampler

	_, err := cmdctx.TryReply("**Sampler set to:** %s", sampler)
	return err
}
