package commands

import (
	"fmt"
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
)

var HelpCommand = command.NewCommand("help", []string{"h", "?"}, helpCommandRun)

func helpCommandRun(cmdctx *command.CommandContext) error {
	content := fmt.Sprintf(`**Usage:** %s<command> [args]
**Commands:** %s`, cmdctx.CalledWithPrefix, strings.Join(cmdctx.Executor.GetCommandNames(), ", "))

	_, err := cmdctx.Executor.SendMessageReply(cmdctx.Message.ChannelID, content, cmdctx.Message.ID)
	if err != nil {
		_, err = cmdctx.Executor.SendMessage(cmdctx.Message.ChannelID, content)
	}

	return err
}
