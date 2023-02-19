package commands

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/kobold"
	"github.com/diamondburned/arikawa/v3/discord"
)

var KoboldCommand = command.NewCommand("kobold", []string{"kb"}, koboldRun)
var ErrKoboldDisabled = errors.New("kobold is disabled")
var KoboldLock = sync.Mutex{}

func koboldRun(cmdctx *command.CommandContext) error {
	if !config.Config.KoboldEnabled {
		return ErrKoboldDisabled
	}

	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Please specify a prompt!**")
		return err
	}

	if !KoboldLock.TryLock() {
		_, err := cmdctx.TryReply("**Kobold is busy, please wait a few seconds!**")
		return err
	}
	defer KoboldLock.Unlock()
	config.ConfigMutex.Lock()
	koboldDM := config.Config.KoboldDMOutput
	config.ConfigMutex.Unlock()

	chID := discord.ChannelID(0)
	msgID := discord.MessageID(0)

	if koboldDM {
		_, _ = cmdctx.TryReply("**Kobold will direct message the response to the sender!**")
		cmdctx.StopTyping <- struct{}{}
		ch, err := cmdctx.Executor.CreatePrivateChannel(cmdctx.Message.Author.ID)
		if err != nil {
			return err
		}
		chID = ch.ID
		msg, err := cmdctx.Executor.SendMessage(chID, ensureLen("**Kobold:** "+cmdctx.Args+" *(Generating...)*"))
		if err != nil {
			return err
		}
		msgID = msg.ID

		stoptyping := make(chan struct{})

		_ = cmdctx.Executor.Typing(chID)
		defer close(stoptyping)
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ticker.C:
					_ = cmdctx.Executor.Typing(chID)
				case <-stoptyping:
					ticker.Stop()
					return
				}
			}
		}()
	}

	data := &kobold.KoboldRequest{
		Prompt:      cmdctx.Args,
		Temperature: 0.7,
		TopP:        1.0,
	}

	res, err := kobold.Generate(data)
	if err != nil {
		log.Println("Could not query kobold:", err)
		return err
	}

	if koboldDM {
		_, err = cmdctx.Executor.EditMessage(chID, msgID, ensureLen("**Kobold:** "+cmdctx.Args+res))
		return err
	} else {
		_, err = cmdctx.TryReply("**Kobold:** %s%s", cmdctx.Args, res)
		return err
	}
}

func ensureLen(str string) string {
	if len(str) > 2000 {
		return str[:2000]
	}
	return str
}
