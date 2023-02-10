package command

import (
	"errors"

	"github.com/ayunami2000/ayunsdcord/utils"
	"github.com/diamondburned/arikawa/v3/state"
)

var ErrCommandNotFound = errors.New("command not found")

type Executor struct {
	*state.State
	commands []*Command
}

func NewExecutor(state *state.State) *Executor {
	return &Executor{State: state}
}

func (e *Executor) GetCommandNames() (names []string) {
	for _, c := range e.commands {
		names = append(names, c.Name)
	}

	return
}

func (e *Executor) RegisterCommand(cmd *Command) {
	e.commands = append(e.commands, cmd)
}

func (e *Executor) RunCommand(name string, cmdctx *CommandContext) error {
	for _, cmd := range e.commands {
		if cmd.Name == name || utils.Contains(cmd.Aliases, name) {
			return cmd.Run(cmdctx)
		}
	}

	return ErrCommandNotFound
}
