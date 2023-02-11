// Adapted from https://github.com/cutest-design/bot2/blob/main/command/command.go (my own code)
package command

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/diamondburned/arikawa/v3/discord"
)

type CurrentRenderInfo struct {
	RequestedBy  discord.UserID
	Task         int64
	LastFrameUrl string
	FrameData    []byte
}

type ChannelSettings struct {
	InUse *atomic.Bool

	Model        string
	VAE          string
	HyperNetwork string

	Prompt         string
	NegativePrompt string

	Width  uint
	Height uint

	PromptStrength float64
	InferenceSteps uint
	GuidanceScale  float64
	Sampler        string
	Upscaler       string
	UpscaleAmount  uint

	CurrentRenderInfo      *CurrentRenderInfo
	CurrentRenderInfoMutex sync.Mutex
	SessionID              string
}

type CommandContext struct {
	Executor        *Executor
	ChannelSettings *ChannelSettings
	Message         *discord.Message

	CalledWithPrefix string
	CalledWithAlias  string
	Args             string
	StopTyping       chan<- struct{}
}

func (c *CommandContext) TryReply(format string, a ...any) (msg *discord.Message, err error) {
	content := fmt.Sprintf(format, a...)
	content = strings.ReplaceAll(content, "@", "@\u200b") // Just in case
	if len(content) > 2000 {
		content = content[:2000]
	}

	msg, err = c.Executor.SendMessageReply(c.Message.ChannelID, content, c.Message.ID)
	if err != nil {
		msg, err = c.Executor.SendMessage(c.Message.ChannelID, content)
	}
	return msg, err
}

type Command struct {
	Name    string
	Aliases []string
	run     func(*CommandContext) error
}

func NewCommand(name string, aliases []string, run func(*CommandContext) error) *Command {
	return &Command{
		Name:    name,
		Aliases: aliases,
		run:     run,
	}
}

func (c *Command) Run(cmdctx *CommandContext) error {
	return c.run(cmdctx)
}

func (c *Command) String() string {
	return fmt.Sprintf("{Name: %s, Aliases: %s}", c.Name, c.Aliases)
}
