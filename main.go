package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ayunami2000/ayunsdcord/commands"
	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/commands/render"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
	"github.com/ayunami2000/ayunsdcord/utils"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
)

func canUse(authorId string) bool {
	config.ConfigMutex.Lock()
	defer config.ConfigMutex.Unlock()
	for _, id := range config.Config.UsersList.List {
		if id == authorId {
			return config.Config.UsersList.WhitelistMode
		}
	}

	return !config.Config.UsersList.WhitelistMode
}

var s *state.State
var botID discord.UserID
var executor *command.Executor
var channels = utils.NewForgetfulMap[string, *command.ChannelSettings](20 * time.Minute)

func messageCreate(c *gateway.MessageCreateEvent) {
	if c.Author.ID == botID {
		return
	}

	config.ConfigMutex.Lock()
	allowbots := config.Config.AllowBots
	config.ConfigMutex.Unlock()

	if c.Author.Bot && !allowbots {
		return
	}

	if len(config.Config.ChannelIds) > 0 &&
		!utils.Contains(config.Config.ChannelIds, c.ChannelID.String()) {
		return
	}

	if !canUse(c.Author.ID.String()) {
		return
	}

	config.ConfigMutex.Lock()
	prefix := config.Config.Prefix
	config.ConfigMutex.Unlock()
	if strings.HasPrefix(c.Content, botID.Mention()) {
		prefix = botID.Mention()
	}

	if !strings.HasPrefix(strings.ToLower(c.Content), prefix) {
		return
	}

	args := strings.Replace(strings.TrimSpace(c.Content[len(prefix):]), "\n", " ", -1)
	if args == "" {
		args = "?"
	}

	settings, settingsInit := channels.Get(c.ChannelID.String())

	if !settingsInit {
		config.ConfigMutex.Lock()
		settings = &command.ChannelSettings{
			Prompt:         config.Config.DefaultPrompt,
			NegativePrompt: config.Config.DefaultNegativePrompt,
			Width:          config.Config.DefaultWidth,
			Height:         config.Config.DefaultHeight,
			PromptStrength: config.Config.DefaultPromptStrength,
			InferenceSteps: config.Config.DefaultInferenceSteps,
			GuidanceScale:  config.Config.DefaultGuidanceScale,
			Sampler:        config.Config.DefaultSampler,
			Upscaler:       config.Config.DefaultUpscaler,
			UpscaleAmount:  config.Config.DefaultUpscaleAmount,
			SessionID:      strconv.Itoa(rand.Int()),
			InUse:          &atomic.Bool{},
		}
		config.ConfigMutex.Unlock()

		appConfig, err := sdapi.GetAppConfig()
		if err != nil {
			_, _ = s.SendMessageReply(c.ChannelID, fmt.Sprintf("**Error:** %v", err), c.ID)
			log.Println("Could not query app config:", err)
			return
		}

		settings.Model = appConfig.Model.StableDiffusion
		settings.VAE = appConfig.Model.VAE
		settings.HyperNetwork = appConfig.Model.HyperNetwork

		channels.Set(c.ChannelID.String(), settings)
	}

	cmd := strings.ToLower(strings.Split(args, " ")[0])
	args = strings.TrimSpace(args[len(cmd):])
	stoptyping := make(chan struct{})
	context := command.CommandContext{
		Executor:         executor,
		ChannelSettings:  settings,
		Message:          &c.Message,
		CalledWithPrefix: prefix,
		CalledWithAlias:  cmd,
		Args:             args,
		StopTyping:       stoptyping,
	}

	_ = s.Typing(c.ChannelID)
	defer close(stoptyping)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				_ = s.Typing(c.ChannelID)
			case <-stoptyping:
				ticker.Stop()
				return
			}
		}
	}()

	if err := executor.RunCommand(cmd, &context); err != nil {
		str := err.Error()
		_, _ = context.TryReply("**Error:** %s.", strings.ToUpper(str[:1])+str[1:])
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	config.ConfigMutex.Lock()
	if config.Config.BotToken == "" {
		log.Fatalln("Missing bot token!")
	}

	if len(config.Config.ChannelIds) < 1 {
		log.Println("Missing channel IDs, will respond in all channels!")
	}

	if config.Config.ImageDumpChannelId == "" {
		log.Println("Missing image dump channel ID, will dump images in the same channel!")
	}

	s = state.New("Bot " + config.Config.BotToken)
	config.ConfigMutex.Unlock()
	s.AddHandler(messageCreate)
	s.AddIntents(gateway.IntentGuildMessages)

	self, err := s.Me()
	if err != nil {
		log.Fatalln("Could not fetch self:", err)
	}

	botID = self.ID
	executor = command.NewExecutor(s)
	executor.RegisterCommand(commands.ClearCommand)
	executor.RegisterCommand(commands.GuidanceScaleCommand)
	executor.RegisterCommand(commands.HelpCommand)
	executor.RegisterCommand(commands.HyperNetworkCommand)
	executor.RegisterCommand(commands.InferenceStepsCommand)
	executor.RegisterCommand(commands.ListModelsCommand)
	executor.RegisterCommand(commands.ModelCommand)
	executor.RegisterCommand(commands.NegativePromptCommand)
	executor.RegisterCommand(commands.PromptCommand)
	executor.RegisterCommand(commands.PromptStrengthCommand)
	executor.RegisterCommand(commands.SamplerCommand)
	executor.RegisterCommand(commands.RandomCommand)
	executor.RegisterCommand(render.RenderCommand)
	executor.RegisterCommand(commands.SizeCommand)
	executor.RegisterCommand(commands.StopCommand)
	executor.RegisterCommand(commands.UpscaleAmountCommand)
	executor.RegisterCommand(commands.UpscalerCommand)
	executor.RegisterCommand(commands.VaeCommand)
	executor.RegisterCommand(commands.KoboldCommand)

	if err := s.Open(context.Background()); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	log.Println("Started as", self.Username)

	config.ConfigMutex.Lock()
	frameUrl := config.Config.FrameUrl
	config.ConfigMutex.Unlock()

	if frameUrl != "" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			split := strings.SplitN(r.RequestURI, "/", 3)
			if len(split) < 3 {
				w.WriteHeader(404)
				return
			}

			channel, channelInit := channels.Get(split[1])
			if !channelInit || channel.CurrentRenderInfo == nil || channel.CurrentRenderInfo.FrameData == nil {
				w.WriteHeader(404)
				return
			}

			w.Header().Add("Content-Type", "image/jpeg")
			w.WriteHeader(200)
			_, _ = w.Write(channel.CurrentRenderInfo.FrameData)
		})

		config.ConfigMutex.Lock()
		frameHttpBind := config.Config.FrameHttpBind
		config.ConfigMutex.Unlock()

		err = http.ListenAndServe(frameHttpBind, nil)

		if err != nil {
			log.Fatalln("Failed to start webserver:", err)
		}
	}

	select {}
}
