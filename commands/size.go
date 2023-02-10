package commands

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/utils"
)

var VALID_SIZES = []uint64{128, 192, 256, 320, 384, 448, 512, 576, 640, 704, 768, 832, 896, 960, 1024, 1280, 1536, 1792, 2048}
var SizeCommand = command.NewCommand("size", []string{"sz"}, sizeRun)
var ErrInvalidSize = errors.New("invalid size")

func parseSize(sz string) (uint, error) {
	i, err := strconv.ParseUint(sz, 10, 64)
	if err != nil {
		return 0, err
	}

	if !utils.Contains(VALID_SIZES, i) {
		return 0, ErrInvalidSize
	}

	return uint(i), nil
}

func parseSizes(sz string) (uint, uint, error) {
	pieces := strings.SplitN(strings.ReplaceAll(strings.ToLower(sz), "x", " "), " ", 2)
	if len(pieces) == 1 {
		i, err := parseSize(pieces[0])
		return i, i, err
	} else {
		width, err := parseSize(pieces[0])
		if err != nil {
			return width, 0, err
		}

		height, err := parseSize(pieces[1])
		return width, height, err
	}
}

func sizeRun(cmdctx *command.CommandContext) error {
	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply(`**Current size:** %dx%d
Sizes: %s`, cmdctx.ChannelSettings.Width, cmdctx.ChannelSettings.Height, strings.Join(utils.ToStringSlice(VALID_SIZES), ", "))
		return err
	}

	if err := config.CanChange(cmdctx.ChannelSettings.InUse, "size"); err != nil {
		return err
	}

	width, height, err := parseSizes(cmdctx.Args)
	if err != nil {
		return err
	}

	cmdctx.ChannelSettings.Width = width
	cmdctx.ChannelSettings.Height = height

	_, err = cmdctx.TryReply("**Size set to:** %dx%d", width, height)
	return err
}
