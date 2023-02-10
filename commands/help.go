package commands

import (
	"strings"

	"github.com/ayunami2000/ayunsdcord/commands/command"
)

var HelpCommand = command.NewCommand("help", []string{"h", "?"}, helpCommandRun)

func helpCommandRun(cmdctx *command.CommandContext) error {
	_, err := cmdctx.TryReply(`**Usage:** %s <command> [args]
**Commands:** %s`, cmdctx.CalledWithPrefix, strings.Join(cmdctx.Executor.GetCommandNames(), ", "))

	return err
}
