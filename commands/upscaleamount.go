package commands

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var VALID_UPSCALE_AMOUNTS = []uint{2, 4}
var UpscaleAmountCommand = command.NewCommand("upscaleamount", []string{"ua"}, upscaleAmountRun)
var ErrInvalidUpscaleAmount = errors.New("invalid upscale amount")

func upscaleAmountRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply(`**Current upscale amount:** %d
Upscale amounts: %s`, cmdctx.ChannelSettings.UpscaleAmount, strings.Join(utils.ToStringSlice(VALID_UPSCALE_AMOUNTS), ", "))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "upscaleamount"); err != nil {
		return err
	}

	i, err := strconv.ParseUint(cmdctx.Args, 10, 64)
	if err != nil {
		return err
	}

	if !utils.Contains(VALID_UPSCALE_AMOUNTS, uint(i)) {
		return ErrInvalidUpscaleAmount
	}

	cmdctx.ChannelSettings.UpscaleAmount = uint(i)
	_, err = cmdctx.TryReply("**Upscale amount set to:** %d", cmdctx.ChannelSettings.UpscaleAmount)
	return err
}
