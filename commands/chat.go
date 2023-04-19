package commands

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/ayunami2000/ayunsdcord/chatapi"
	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/diamondburned/arikawa/v3/discord"
)

var ChatCommand = command.NewCommand("chat", []string{"ch"}, chatRun)
var ErrChatDisabled = errors.New("chat is disabled")
var ChatLock = sync.Mutex{}

func chatRun(cmdctx *command.CommandContext) error {
	if !config.Config.ChatEnabled {
		return ErrChatDisabled
	}

	if cmdctx.Args == "" {
		_, err := cmdctx.TryReply("**Please specify a prompt!**")
		return err
	}

	if !ChatLock.TryLock() {
		_, err := cmdctx.TryReply("**Chat is busy, please wait a few seconds!**")
		return err
	}
	defer ChatLock.Unlock()
	config.ConfigMutex.Lock()
	chatDM := config.Config.ChatDMOutput
	config.ConfigMutex.Unlock()

	chID := discord.ChannelID(0)
	msgID := discord.MessageID(0)

	if chatDM {
		_, _ = cmdctx.TryReply("**Chat will direct message the response to the sender!**")
		cmdctx.StopTyping <- struct{}{}
		ch, err := cmdctx.Executor.CreatePrivateChannel(cmdctx.Message.Author.ID)
		if err != nil {
			return err
		}
		chID = ch.ID
		msg, err := cmdctx.Executor.SendMessage(chID, ensureLen("*(Generating...)*"))
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
	} else {
		msg, err := cmdctx.TryReply("*(Generating...)*")
		if err != nil {
			return err
		}
		chID = msg.ChannelID
		msgID = msg.ID
	}

	res, err := chatapi.Generate(cmdctx.Args)
	if err != nil {
		log.Println("Could not query chat:", err)
		return err
	}

	_, err = cmdctx.Executor.EditMessage(chID, msgID, ensureLen(res))
	return err
}

func ensureLen(str string) string {
	if len(str) > 2000 {
		return str[:2000]
	}
	return str
}
