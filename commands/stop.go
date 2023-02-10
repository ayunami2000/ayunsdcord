package commands

import (
	"errors"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/sdapi"
)

var ErrRenderNotInProgress = errors.New("no render in progress")
var ErrRenderNotRequestedByYou = errors.New("current render not requested by you")
var StopCommand = command.NewCommand("stop", []string{"s"}, stopRun)

func stopRun(cmdctx *command.CommandContext) error {
	if !cmdctx.ChannelSettings.InUse.Load() {
		return ErrRenderNotInProgress
	}

	cmdctx.ChannelSettings.CurrentRenderInfoMutex.Lock()
	defer cmdctx.ChannelSettings.CurrentRenderInfoMutex.Unlock()

	renderinfo := cmdctx.ChannelSettings.CurrentRenderInfo
	if renderinfo == nil || renderinfo.Task == 0 {
		return ErrRenderNotInProgress
	}

	if renderinfo.RequestedBy != cmdctx.Message.Author.ID {
		return ErrRenderNotRequestedByYou
	}

	if err := sdapi.StopRender(renderinfo.Task); err != nil {
		return err
	}

	_, err := cmdctx.TryReply("**Stopped current render**")
	return err
}
