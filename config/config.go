package config

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var ErrCannotChangeProperty = errors.New("not allowed to change property")
var ErrPropertyLocked = errors.New("locked while rendering")
var imageDumpChannelId = discord.NullChannelID

type UsersList struct {
	WhitelistMode bool
	List          []string
}

type configStruct struct {
	BotToken           string
	ChannelIds         []string
	ImageDumpChannelId string
	Prefix             string
	AllowBots          bool

	StableDiffusionURL  string
	BasicAuth           string
	StreamImageProgress uint

	FrameUrl        string
	FrameHttpBind   string
	CountFrameless  bool
	LoadingFrameUrl string
	ErrorFrameUrl   string

	DefaultPrompt         string
	DefaultNegativePrompt string

	DefaultWidth  uint
	DefaultHeight uint

	DefaultPromptStrength float64
	DefaultInferenceSteps uint
	DefaultGuidanceScale  float64
	DefaultSampler        string
	DefaultUpscaler       string
	DefaultUpscaleAmount  uint

	DenyChanging []string
	UsersList    UsersList
}

var Config = configStruct{}
var ConfigMutex = sync.Mutex{}

func init() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AutomaticEnv()

	viper.SetDefault("Prefix", "sd!")
	viper.SetDefault("ChannelIds", []string{})
	viper.SetDefault("AllowBots", false)

	viper.SetDefault("StableDiffusionURL", "http://localhost:9000")
	viper.SetDefault("StreamImageProgress", 5)
	viper.SetDefault("CountFrameless", false)

	viper.SetDefault("FrameHttpBind", ":8080")
	viper.SetDefault("FrameUrl", "")
	viper.SetDefault("LoadingFrameUrl", "https://c.tenor.com/RVvnVPK-6dcAAAAC/reload-cat.gif")
	viper.SetDefault("ErrorFrameUrl", "https://upload.wikimedia.org/wikipedia/commons/f/f7/Generic_error_message.png")

	viper.SetDefault("DefaultPrompt", "cat")
	viper.SetDefault("DefaultNegativePrompt", "nsfw")

	viper.SetDefault("DefaultWidth", 768)
	viper.SetDefault("DefaultHeight", 768)

	viper.SetDefault("DefaultPromptStrength", 0.8)
	viper.SetDefault("DefaultInferenceSteps", 28)
	viper.SetDefault("DefaultGuidanceScale", 12.0)
	viper.SetDefault("DefaultSampler", "euler_a")
	viper.SetDefault("DefaultUpscaleAmount", 2)

	viper.SetDefault("DenyChanging", []string{})
	viper.SetDefault("UsersList.WhitelistMode", false)
	viper.SetDefault("UsersList.List", []string{})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("Unable to open config file: %v\n", err)
		}
	} else {
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Fatalf("Unable to write to config file: %v\n", err)
			}
		}
	}

	err := viper.Unmarshal(&Config)
	if err != nil {
		log.Fatalf("Unable to decode config: %v\n", err)
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		newConfig := configStruct{}
		err := viper.Unmarshal(&newConfig)
		if err == nil {
			ConfigMutex.Lock()
			Config = newConfig
			ConfigMutex.Unlock()
			log.Println("Successfully updated config")
		}
	})
}

func GetImageDumpChannelId() discord.ChannelID {
	if Config.ImageDumpChannelId == "" {
		return discord.NullChannelID
	} else if imageDumpChannelId != discord.NullChannelID {
		return imageDumpChannelId
	}

	i, err := strconv.ParseUint(Config.ImageDumpChannelId, 10, 64)
	if err != nil {
		log.Fatalln("Invalid image dump channel ID!")
	}

	imageDumpChannelId = discord.ChannelID(i)
	return imageDumpChannelId
}

func CanChange_NoLock(s string) bool {
	ConfigMutex.Lock()
	defer ConfigMutex.Unlock()
	for _, v := range Config.DenyChanging {
		if strings.EqualFold(strings.ReplaceAll(v, "_", ""), s) {
			return false
		}
	}

	return true
}

func CanChange(b *atomic.Bool, s string) error {
	if !CanChange_NoLock(s) {
		return ErrCannotChangeProperty
	}

	if b.Load() {
		return ErrPropertyLocked
	}

	return nil
}
