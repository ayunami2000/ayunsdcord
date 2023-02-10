package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
)

var ClearCommand = command.NewCommand("clear", []string{"cl"}, clearCommandRun)
var ErrCannotChangeProperty = errors.New("not allowed to change property")
var ErrInvalidProperty = errors.New("invalid property specified")

var chgMap = map[string]string{
	"p":  "prompt",
	"np": "negativeprompt",
	"ps": "promptstrength",
	"is": "inferencesteps",
	"gs": "guidancescale",
	"sz": "size",
	"v":  "vae",
	"hn": "hypernetwork",
	"u":  "upscaler",
	"ua": "upscaleamount",
}

func clearCommandRun(cmdctx *command.CommandContext) error {
	args := strings.ToLower(cmdctx.Args)
	if args == "" {
		valid := []string{}

		for k, v := range chgMap {
			valid = append(valid, fmt.Sprintf("%s/%s", k, v))
		}

		return fmt.Errorf("%w, valid properties: %s", ErrInvalidProperty, strings.Join(valid, ", "))
	}

	mappedArg, exists := chgMap[args]
	if !exists {
		mappedArg = args
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, cmdctx.Args); err != nil {
		return ErrCannotChangeProperty
	}

	config.ConfigMutex.Lock()
	switch mappedArg {
	case "prompt":
		cmdctx.ChannelSettings.Prompt = ""
	case "negativeprompt":
		cmdctx.ChannelSettings.NegativePrompt = ""
	case "promptstrength":
		cmdctx.ChannelSettings.PromptStrength = config.Config.DefaultPromptStrength
	case "inferencesteps":
		cmdctx.ChannelSettings.InferenceSteps = config.Config.DefaultInferenceSteps
	case "guidancescale":
		cmdctx.ChannelSettings.GuidanceScale = config.Config.DefaultGuidanceScale
	case "size":
		cmdctx.ChannelSettings.Width = config.Config.DefaultWidth
		cmdctx.ChannelSettings.Height = config.Config.DefaultHeight
	case "vae":
		cmdctx.ChannelSettings.VAE = ""
	case "hypernetwork":
		cmdctx.ChannelSettings.HyperNetwork = ""
	case "upscaler":
		cmdctx.ChannelSettings.Upscaler = ""
	case "upscaleamount":
		cmdctx.ChannelSettings.UpscaleAmount = 0
	default:
		config.ConfigMutex.Unlock()
		return ErrInvalidProperty
	}
	config.ConfigMutex.Unlock()

	_, err := cmdctx.TryReply("**Successfully cleared property**")
	return err
}
