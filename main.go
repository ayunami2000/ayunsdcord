package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/tjarratt/babble"
)

type Config struct {
	BotToken                    string   `json:"bot_token,omitempty"`
	StableDiffusionURL          string   `json:"sd_url,omitempty"`
	BasicAuth                   string   `json:"basic_auth,omitempty"`
	ChannelIds                  []string `json:"channel_ids,omitempty"`
	ImageDumpChannelId          string   `json:"image_dump_channel_id,omitempty"`
	Prefix                      string   `json:"prefix,omitempty"`
	FrameUrl                    string   `json:"frame_url,omitempty"`
	FrameHttpBind               string   `json:"frame_http_bind,omitempty"`
	AllowBots                   bool     `json:"allow_bots,omitempty"`
	DefaultPrompt               string   `json:"default_prompt,omitempty"`
	DefaultNegativePrompt       string   `json:"default_negative_prompt,omitempty"`
	DefaultWidth                int      `json:"default_width,omitempty"`
	DefaultHeight               int      `json:"default_height,omitempty"`
	DefaultInferenceSteps       int      `json:"default_inference_steps,omitempty"`
	DefaultGuidanceScale        float64  `json:"default_guidance_scale,omitempty"`
	AllowChangingNegativePrompt bool     `json:"allow_changing_negative_prompt,omitempty"`
	StreamImageProgress         bool     `json:"stream_image_progress,omitempty"`
	LoadingFrameUrl             string   `json:"loading_frame_url,omitempty"`
	ErrorFrameUrl               string   `json:"error_frame_url,omitempty"`
	CountFrameless              bool     `json:"count_frameless,omitempty"`
	DefaultPromptStrength       float64  `json:"default_prompt_strength,omitempty"`
	AllowChangingSize           bool     `json:"allow_changing_size,omitempty"`
}

type UsersList struct {
	WhitelistMode bool     `json:"whitelist_mode"`
	List          []string `json:"list"`
}

type Render struct {
	Prompt                  string   `json:"prompt"`
	Seed                    int      `json:"seed"`
	NegativePrompt          string   `json:"negative_prompt"`
	NumOutputs              int      `json:"num_outputs"`
	NumInferenceSteps       int      `json:"num_inference_steps"`
	GuidanceScale           float64  `json:"guidance_scale"`
	Width                   int      `json:"width"`
	Height                  int      `json:"height"`
	VramUsageLevel          string   `json:"vram_usage_level"`
	UseStableDiffusionModel string   `json:"use_stable_diffusion_model"`
	UseVaeModel             string   `json:"use_vae_model,omitempty"`
	UseHypernetworkModel    string   `json:"use_hypernetwork_model,omitempty"`
	StreamProgressUpdates   bool     `json:"stream_progress_updates"`
	StreamImageProgress     bool     `json:"stream_image_progress"`
	ShowOnlyFilteredImage   bool     `json:"show_only_filtered_image"`
	OutputFormat            string   `json:"output_format"`
	OutputQuality           int      `json:"output_quality"`
	MetadataOutputFormat    string   `json:"metadata_output_format"`
	OriginalPrompt          string   `json:"original_prompt"`
	ActiveTags              []string `json:"active_tags"`
	InactiveTags            []string `json:"inactive_tags"`
	SamplerName             string   `json:"sampler_name"`
	SessionId               string   `json:"session_id"`
	InitImage               string   `json:"init_image,omitempty"`
	PromptStrength          float64  `json:"prompt_strength,omitempty"`
}

type RenderResponse struct {
	Stream string `json:"stream"`
}

type ModelsResponse struct {
	Options struct {
		StableDiffusion []string `json:"stable-diffusion"`
		VAE             []string `json:"vae"`
		HyperNetwork    []string `json:"hypernetwork"`
	} `json:"options"`
}

type AppConfigResponse struct {
	Model struct {
		StableDiffusion string `json:"stable-diffusion"`
		VAE             string `json:"vae,omitempty"`
		HyperNetwork    string `json:"hypernetwork,omitempty"`
	} `json:"model"`
}

type StreamResponse struct {
	Output []struct {
		Path string `json:"path,omitempty"`
		Data string `json:"data,omitempty"`
	} `json:"output"`
	Step       int    `json:"step,omitempty"`
	TotalSteps int    `json:"total_steps,omitempty"`
	Status     string `json:"status,omitempty"`
}

type Settings struct {
	InUse          sync.Mutex
	Model          string
	VAE            string
	HyperNetwork   string
	Prompt         string
	NegativePrompt string
	Width          int
	Height         int
	PromptStrength float64
	InferenceSteps int
	GuidanceScale  float64
	LastFrameUrl   string
	FrameData      []byte
}

// https://stackoverflow.com/a/40326580/6917520
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseBool(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		log.Fatalf("Error parsing bool: %v", err)
	}
	return b
}

func parseInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		log.Fatalf("Error parsing int: %v", err)
	}
	return i
}

func parseFloat(str string) float64 {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Fatalf("Error parsing float: %v", err)
	}
	return f
}

func floatToStr(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}

var config = Config{
	BotToken:                    getEnv("BOT_TOKEN", ""),
	StableDiffusionURL:          getEnv("SD_URL", "http://localhost:9000"),
	BasicAuth:                   getEnv("BASIC_AUTH", ""),
	ChannelIds:                  strings.Split(getEnv("CHANNEL_IDS", ""), ","),
	ImageDumpChannelId:          getEnv("IMAGE_DUMP_CHANNEL_ID", ""),
	Prefix:                      getEnv("PREFIX", "sd!"),
	FrameUrl:                    getEnv("FRAME_URL", ""),
	FrameHttpBind:               getEnv("FRAME_HTTP_BIND", ":8080"),
	AllowBots:                   parseBool(getEnv("ALLOW_BOTS", "false")),
	DefaultPrompt:               getEnv("DEFAULT_PROMPT", "cat"),
	DefaultNegativePrompt:       getEnv("DEFAULT_NEGATIVE_PROMPT", "nsfw"),
	DefaultWidth:                parseInt(getEnv("DEFAULT_WIDTH", "768")),
	DefaultHeight:               parseInt(getEnv("DEFAULT_HEIGHT", "768")),
	AllowChangingNegativePrompt: parseBool(getEnv("ALLOW_CHANGING_NEGATIVE_PROMPT", "true")),
	StreamImageProgress:         parseBool(getEnv("STREAM_IMAGE_PROGRESS", "true")),
	LoadingFrameUrl:             getEnv("LOADING_FRAME_URL", "https://c.tenor.com/RVvnVPK-6dcAAAAC/reload-cat.gif"),
	ErrorFrameUrl:               getEnv("ERROR_FRAME_URL", "https://upload.wikimedia.org/wikipedia/commons/f/f7/Generic_error_message.png"),
	CountFrameless:              parseBool(getEnv("COUNT_FRAMELESS", "false")),
	DefaultPromptStrength:       parseFloat(getEnv("DEFAULT_PROMPT_STRENGTH", "0.8")),
	DefaultInferenceSteps:       parseInt(getEnv("DEFAULT_INFERENCE_STEPS", "28")),
	DefaultGuidanceScale:        parseFloat(getEnv("DEFAULT_GUIDANCE_SCALE", "12.0")),
	AllowChangingSize:           parseBool(getEnv("ALLOW_CHANGING_SIZE", "true")),
}

var usersList = UsersList{
	WhitelistMode: parseBool(getEnv("WHITELIST_MODE", "false")),
	List:          strings.Split(getEnv("USERS_LIST", ""), ","),
}

func millisStr() string {
	return strconv.Itoa(int(time.Now().UnixMilli()))
}

var s *state.State
var botID discord.UserID
var ctx context.Context
var channels = make(map[string]*Settings)
var sessionId string = millisStr()
var babbler babble.Babbler
var imageDumpChannelId discord.ChannelID

func Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if config.BasicAuth != "" {
		req.Header.Set("Authorization", "Basic "+config.BasicAuth)
	}
	return http.DefaultClient.Do(req)
}

func Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if config.BasicAuth != "" {
		req.Header.Set("Authorization", "Basic "+config.BasicAuth)
	}
	return http.DefaultClient.Do(req)
}

func reply(channelId discord.ChannelID, referenceId discord.MessageID, message string) (*discord.Message, error) {
	msg, err := s.SendMessageReply(channelId, message, referenceId)
	if err != nil {
		msg, err = s.SendMessage(channelId, message)
	}
	return msg, err
}

func frame(channelId discord.ChannelID, referenceId discord.MessageID, reader io.Reader, step int, totalSteps int, hasInitImage bool) (*discord.Message, error) {
	channel := channels[channelId.String()]
	if reader == nil {
		frameEmbed(channelId, referenceId, "", step, totalSteps, hasInitImage)
		return nil, nil
	}
	if config.FrameUrl != "" && step != totalSteps {
		channel.FrameData, _ = io.ReadAll(reader)
		frameEmbed(channelId, referenceId, config.FrameUrl+"/"+channelId.String()+"/"+millisStr(), step, totalSteps, hasInitImage)
		return nil, nil
	}
	dumpChannel := channelId
	if config.ImageDumpChannelId != "" {
		dumpChannel = imageDumpChannelId
	}
	msg, err := s.SendMessageComplex(dumpChannel, api.SendMessageData{
		Files: []sendpart.File{{
			Name:   "stable-diffusion_" + millisStr() + ".png",
			Reader: reader,
		}},
	})
	if err != nil {
		s.EditMessage(channelId, referenceId, "failed to upload progress image")
	} else {
		frameEmbed(channelId, referenceId, msg.Attachments[0].URL, step, totalSteps, hasInitImage)
	}
	return msg, err
}

func frameEmbed(channelId discord.ChannelID, referenceId discord.MessageID, url string, step int, totalSteps int, hasInitImage bool) {
	channel := channels[channelId.String()]
	footer := "Done!"
	if step == 0 && totalSteps == 0 {
		footer = "Error."
	}
	if step != totalSteps {
		footer = "Step " + strconv.Itoa(step) + " of " + strconv.Itoa(totalSteps)
	}
	if url != "" {
		channel.LastFrameUrl = url
	}
	desc := "**Prompt:** " + channel.Prompt
	if channel.NegativePrompt != "" {
		desc += "\n**Negative Prompt:** " + channel.NegativePrompt
	}
	desc += "\n**Width:** " + strconv.Itoa(channel.Width) + "\n**Height:** " + strconv.Itoa(channel.Height) + "\n**Inference Steps:** " + strconv.Itoa(channel.InferenceSteps) + "\n**Guidance Scale:** " + floatToStr(channel.GuidanceScale) + "\n**Model:** " + channel.Model
	if channel.VAE != "" {
		desc += "\n**VAE:** " + channel.VAE
	}
	if channel.HyperNetwork != "" {
		desc += "\n**HyperNetwork:** " + channel.HyperNetwork
	}
	if hasInitImage {
		desc += "\n**Img2Img Prompt Strength:** " + floatToStr(channel.PromptStrength)
	}
	s.EditMessageComplex(channelId, referenceId, api.EditMessageData{
		Content: option.NewNullableString(""),
		Embeds: &[]discord.Embed{{
			Title:       "Stable Diffusion",
			Description: desc,
			Footer: &discord.EmbedFooter{
				Text: footer,
			},
			Image: &discord.EmbedImage{
				URL: channel.LastFrameUrl,
			},
			Timestamp: discord.NewTimestamp(time.Now()),
		}},
	})
}

func getModels() (*ModelsResponse, error) {
	res, err := Get(config.StableDiffusionURL + "/get/models")
	if err != nil {
		return nil, err
	} else {
		resParsed := new(ModelsResponse)
		json.NewDecoder(res.Body).Decode(resParsed)
		res.Body.Close()
		return resParsed, nil
	}
}

func randomPrompt(args string) (string, error) {
	if args == "" {
		babbler.Count = 10
	} else {
		i, err := strconv.Atoi(args)
		if err != nil {
			return "", err
		} else {
			if i < 1 {
				i = 1
			} else if i > 100 {
				i = 100
			}
			babbler.Count = i
		}
	}
	return babbler.Babble(), nil
}

// https://stackoverflow.com/a/59955447/6917520
func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndexAny(s[:max], " .,:;-")]
}

func canUse(authorId string) bool {
	for _, id := range usersList.List {
		if id == authorId {
			return usersList.WhitelistMode
		}
	}
	return !usersList.WhitelistMode
}

func setCurrentModels(channelId string) {
	channel := channels[channelId]
	res, err := Get(config.StableDiffusionURL + "/get/app_config")
	if err != nil {
		log.Fatalln("Failed to get active models!")
	} else {
		resParsed := new(AppConfigResponse)
		json.NewDecoder(res.Body).Decode(resParsed)
		channel.Model = resParsed.Model.StableDiffusion
		channel.VAE = resParsed.Model.VAE
		channel.HyperNetwork = resParsed.Model.HyperNetwork
		res.Body.Close()
	}
}

func messageCreate(c *gateway.MessageCreateEvent) {
	if c.Author.ID == botID {
		return
	}

	if c.Author.Bot && !config.AllowBots {
		return
	}

	if len(config.ChannelIds) > 0 {
		pleaseExit := true
		for _, channelId := range config.ChannelIds {
			if channelId == c.ChannelID.String() {
				pleaseExit = false
				break
			}
		}
		if pleaseExit {
			return
		}
	}

	if !canUse(c.Author.ID.String()) {
		return
	}

	prefix := config.Prefix

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

	channel, channelInit := channels[c.ChannelID.String()]

	if !channelInit {
		channels[c.ChannelID.String()] = &Settings{
			Prompt:         config.DefaultPrompt,
			NegativePrompt: config.DefaultNegativePrompt,
			Width:          config.DefaultWidth,
			Height:         config.DefaultHeight,
			PromptStrength: config.DefaultPromptStrength,
			InferenceSteps: config.DefaultInferenceSteps,
			GuidanceScale:  config.DefaultGuidanceScale,
		}
		setCurrentModels(c.ChannelID.String())
		channel = channels[c.ChannelID.String()]
	}

	if !channel.InUse.TryLock() {
		reply(c.ChannelID, c.ID, "**Error:** Already in use.")
		return
	}

	defer channel.InUse.Unlock()

	if err := s.Typing(c.ChannelID); err != nil {
		log.Println("Could not start typing:", err)
	}

	stoptyping := make(chan struct{})
	defer close(stoptyping)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				s.Typing(c.ChannelID)
			case <-stoptyping:
				ticker.Stop()
				return
			}
		}
	}()

	cmd := strings.ToLower(strings.Split(args, " ")[0])
	theRest := strings.TrimSpace(args[len(cmd):])

	if cmd != "render" && cmd != "r" && cmd != "randomrender" && cmd != "rr" {
		if cmd == "help" || cmd == "h" || cmd == "?" {
			reply(c.ChannelID, c.ID, "**Usage:** "+prefix+" (command) [args]\n**Commands:** listmodels, model, vae, hypernetwork, clear, render, size, promptstrength, inferencesteps, guidancescale, random, randomrender, help")
		} else if cmd == "random" || cmd == "rand" {
			pr, err := randomPrompt(theRest)
			if err != nil {
				reply(c.ChannelID, c.ID, "**Error:** Invalid number!")
			} else {
				channel.Prompt = truncateText(pr, 512)
				reply(c.ChannelID, c.ID, "**Prompt randomly set to:** "+channel.Prompt)
			}
		} else if cmd == "listmodels" || cmd == "lm" {
			res, err := getModels()
			if err != nil {
				reply(c.ChannelID, c.ID, "**Error:** Failed to get models!")
			} else {
				reply(c.ChannelID, c.ID, "**Models:**\n__Stable Diffusion__: "+strings.Join(res.Options.StableDiffusion, ", ")+"\n__VAE__: "+strings.Join(res.Options.VAE, ", ")+"\n__HyperNetwork__: "+strings.Join(res.Options.HyperNetwork, ", "))
			}
		} else if cmd == "clear" || cmd == "cl" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Error:** Please specify which parameter to clear/reset: prompt, negativeprompt, model, vae, hypernetwork, size, promptstrength")
			} else if strings.EqualFold(theRest, "prompt") || strings.EqualFold(theRest, "p") {
				channel.Prompt = ""
				reply(c.ChannelID, c.ID, "**Cleared the prompt!**")
			} else if strings.EqualFold(theRest, "negativeprompt") || strings.EqualFold(theRest, "np") {
				if config.AllowChangingNegativePrompt {
					channel.NegativePrompt = ""
					reply(c.ChannelID, c.ID, "**Cleared the negative prompt!**")
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Changing the negative prompt is disabled!")
				}
			} else if strings.EqualFold(theRest, "model") || strings.EqualFold(theRest, "m") {
				reply(c.ChannelID, c.ID, "**Error:** The Model cannot be cleared!")
			} else if strings.EqualFold(theRest, "promptstrength") || strings.EqualFold(theRest, "ps") {
				channel.PromptStrength = config.DefaultPromptStrength
				reply(c.ChannelID, c.ID, "**Reset the Img2Img prompt strength to:** "+floatToStr(channel.PromptStrength))
			} else if strings.EqualFold(theRest, "inferencesteps") || strings.EqualFold(theRest, "is") {
				channel.InferenceSteps = config.DefaultInferenceSteps
				reply(c.ChannelID, c.ID, "**Reset the inference steps to:** "+strconv.Itoa(channel.InferenceSteps))
			} else if strings.EqualFold(theRest, "guidancescale") || strings.EqualFold(theRest, "gs") {
				channel.GuidanceScale = config.DefaultGuidanceScale
				reply(c.ChannelID, c.ID, "**Reset the guidance scale to:** "+floatToStr(channel.GuidanceScale))
			} else if strings.EqualFold(theRest, "size") || strings.EqualFold(theRest, "sz") {
				if config.AllowChangingSize {
					channel.Width = config.DefaultWidth
					channel.Height = config.DefaultHeight
					reply(c.ChannelID, c.ID, "**Reset the size to:** "+strconv.Itoa(channel.Width)+"x"+strconv.Itoa(channel.Height))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Changing the size is disabled!")
				}
			} else if strings.EqualFold(theRest, "vae") || strings.EqualFold(theRest, "v") {
				channel.VAE = ""
				reply(c.ChannelID, c.ID, "**Cleared the VAE!**")
			} else if strings.EqualFold(theRest, "hypernetwork") || strings.EqualFold(theRest, "hypnet") || strings.EqualFold(theRest, "hn") {
				channel.HyperNetwork = ""
				reply(c.ChannelID, c.ID, "**Cleared the HyperNetwork!**")
			} else {
				reply(c.ChannelID, c.ID, "**Error:** Invalid parameter to clear!")
			}
		} else if cmd == "model" || cmd == "m" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current Model:** "+channel.Model)
			} else {
				res, err := getModels()
				if err != nil {
					reply(c.ChannelID, c.ID, "**Error:** Failed to check models!")
				} else {
					found := false
					for _, m := range res.Options.StableDiffusion {
						if strings.EqualFold(m, theRest) {
							found = true
							break
						}
					}
					if found {
						channel.Model = theRest
						reply(c.ChannelID, c.ID, "**Model set to:** "+channel.Model)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid model!")
					}
				}
			}
		} else if cmd == "vae" || cmd == "v" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current VAE:** "+channel.VAE)
			} else {
				res, err := getModels()
				if err != nil {
					reply(c.ChannelID, c.ID, "**Error:** Failed to check models!")
				} else {
					found := false
					for _, m := range res.Options.VAE {
						if strings.EqualFold(m, theRest) {
							found = true
							break
						}
					}
					if found {
						channel.VAE = theRest
						reply(c.ChannelID, c.ID, "**VAE set to:** "+channel.VAE)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid VAE!")
					}
				}
			}
		} else if cmd == "hypernetwork" || cmd == "hypnet" || cmd == "hn" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current HyperNetwork:** "+channel.HyperNetwork)
			} else {
				res, err := getModels()
				if err != nil {
					reply(c.ChannelID, c.ID, "**Error:** Failed to check models!")
				} else {
					found := false
					for _, m := range res.Options.HyperNetwork {
						if strings.EqualFold(m, theRest) {
							found = true
							break
						}
					}
					if found {
						channel.HyperNetwork = theRest
						reply(c.ChannelID, c.ID, "**HyperNetwork set to:** "+channel.HyperNetwork)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid HyperNetwork!")
					}
				}
			}
		} else if cmd == "prompt" || cmd == "p" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current prompt:** "+channel.Prompt)
			} else {
				channel.Prompt = truncateText(theRest, 512)
				reply(c.ChannelID, c.ID, "**Prompt set to:** "+channel.Prompt)
			}
		} else if cmd == "negativeprompt" || cmd == "np" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current negative prompt:** "+channel.NegativePrompt)
			} else {
				if config.AllowChangingNegativePrompt {
					channel.NegativePrompt = truncateText(theRest, 512)
					reply(c.ChannelID, c.ID, "**Negative prompt set to:** "+channel.NegativePrompt)
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Changing the negative prompt is disabled!")
				}
			}
		} else if cmd == "promptstrength" || cmd == "ps" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current Img2Img prompt strength:** "+floatToStr(channel.PromptStrength))
			} else {
				if f, err := strconv.ParseFloat(theRest, 64); err == nil {
					if f < 0 {
						f = 0
					} else if f > 0.999999 {
						f = 0.999999
					}
					channel.PromptStrength = f
					reply(c.ChannelID, c.ID, "**Img2Img prompt strength set to:** "+floatToStr(channel.PromptStrength))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Invalid Img2Img prompt strength!")
				}
			}
		} else if cmd == "inferencesteps" || cmd == "is" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current inference steps:** "+floatToStr(float64(channel.InferenceSteps)))
			} else {
				if i, err := strconv.Atoi(theRest); err == nil {
					if i < 1 {
						i = 1
					} else if i > 100 {
						i = 100
					}
					channel.InferenceSteps = i
					reply(c.ChannelID, c.ID, "**Inference steps set to:** "+strconv.Itoa(channel.InferenceSteps))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Invalid inference steps!")
				}
			}
		} else if cmd == "guidancescale" || cmd == "gs" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current guidance scale:** "+floatToStr(channel.GuidanceScale))
			} else {
				if f, err := strconv.ParseFloat(theRest, 64); err == nil {
					if f < 1.1 {
						f = 1.1
					} else if f > 50 {
						f = 50
					}
					channel.GuidanceScale = f
					reply(c.ChannelID, c.ID, "**Guidance scale set to:** "+floatToStr(channel.GuidanceScale))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Invalid guidance scale!")
				}
			}
		} else if cmd == "size" || cmd == "sz" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current size:** "+strconv.Itoa(channel.Width)+"x"+strconv.Itoa(channel.Height)+"\n**Sizes:** 0: 768x768, 1: 1280x768, 2: 768x1280, 3: 512x512, 4: 896x512, 5: 512x896")
			} else {
				if config.AllowChangingSize {
					if theRest == "0" {
						channel.Width = 768
						channel.Height = 768
					} else if theRest == "1" {
						channel.Width = 1280
						channel.Height = 768
					} else if theRest == "2" {
						channel.Width = 768
						channel.Height = 1280
					} else if theRest == "3" {
						channel.Width = 512
						channel.Height = 512
					} else if theRest == "4" {
						channel.Width = 896
						channel.Height = 512
					} else if theRest == "5" {
						channel.Width = 512
						channel.Height = 896
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid size!")
						return
					}
					reply(c.ChannelID, c.ID, "**Size set to:** "+strconv.Itoa(channel.Width)+"x"+strconv.Itoa(channel.Height))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Changing the size is disabled!")
				}
			}
		} else {
			reply(c.ChannelID, c.ID, "**Error:** Unknown command!")
		}
		return
	}

	if cmd == "randomrender" || cmd == "rr" {
		pr, err := randomPrompt(theRest)
		if err != nil {
			reply(c.ChannelID, c.ID, "**Error:** Invalid number!")
		} else {
			channel.Prompt = truncateText(pr, 512)
			reply(c.ChannelID, c.ID, "**Prompt randomly set to:** "+channel.Prompt)
		}
	} else if theRest != "" {
		channel.Prompt = truncateText(theRest, 512)
		reply(c.ChannelID, c.ID, "**Prompt set to:** "+channel.Prompt)
	}

	body := &Render{
		Prompt:                  channel.Prompt,
		Seed:                    rand.Intn(1000000),
		NegativePrompt:          channel.NegativePrompt,
		NumOutputs:              1,
		NumInferenceSteps:       channel.InferenceSteps,
		GuidanceScale:           channel.GuidanceScale,
		Width:                   channel.Width,
		Height:                  channel.Height,
		VramUsageLevel:          "high",
		UseStableDiffusionModel: channel.Model,
		StreamProgressUpdates:   true,
		StreamImageProgress:     config.StreamImageProgress,
		ShowOnlyFilteredImage:   true,
		OutputFormat:            "png",
		OutputQuality:           75,
		MetadataOutputFormat:    "txt",
		OriginalPrompt:          channel.Prompt,
		ActiveTags:              []string{},
		InactiveTags:            []string{},
		SamplerName:             "euler_a", // dpmpp_2m
		SessionId:               sessionId,
	}

	if channel.VAE != "" {
		body.UseVaeModel = channel.VAE
	}
	if channel.HyperNetwork != "" {
		body.UseHypernetworkModel = channel.HyperNetwork
	}

	if len(c.Attachments) > 0 && strings.HasPrefix(c.Attachments[0].ContentType, "image/") {
		res, err := http.Get(c.Attachments[0].URL)
		if err == nil {
			img, _ := io.ReadAll(res.Body)
			body.InitImage = "data:" + c.Attachments[0].ContentType + ";base64," + base64.StdEncoding.EncodeToString(img)
			body.PromptStrength = channel.PromptStrength
			res.Body.Close()
			reply(c.ChannelID, c.ID, "**Loaded Img2Img image from attachment!**")
			if c.Attachments[0].Description != "" {
				if f, err := strconv.ParseFloat(c.Attachments[0].Description, 64); err == nil {
					if f < 0 {
						f = 0
					} else if f > 0.999999 {
						f = 0.999999
					}
					channel.PromptStrength = f
					reply(c.ChannelID, c.ID, "**Loaded Img2Img prompt strength from alt text:** "+floatToStr(channel.PromptStrength))
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Invalid Img2Img prompt strength in alt text!")
				}
			}
		} else {
			reply(c.ChannelID, c.ID, "**Error:** Failed to download image for Img2Img!")
		}
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(body)
	res, err := Post(config.StableDiffusionURL+"/render", "application/json", buf)

	if err != nil {
		log.Println("Could not query stable diffusion ui:", err)
		reply(c.ChannelID, c.ID, "Failed to query stable diffusion ui")
		return
	}

	resParsed := new(RenderResponse)

	json.NewDecoder(res.Body).Decode(resParsed)

	res.Body.Close()

	msg, err := reply(c.ChannelID, c.ID, "**Loading...**")

	if err != nil {
		return
	}

	if err := s.Typing(c.ChannelID); err != nil {
		log.Println("Could not start typing:", err)
	}

	doneRendering := bool(false)

	lastFrame := new(discord.Message)
	channel.LastFrameUrl = config.LoadingFrameUrl

	step := -1
	totalSteps := 28

	stillTyping := true

	for !doneRendering {
		res, err = Get(config.StableDiffusionURL + resParsed.Stream)

		if err != nil {
			log.Println("Could not query stable diffusion ui:", err)
			s.EditMessage(c.ChannelID, msg.ID, "Failed to query stable diffusion ui")
			return
		}

		dec := json.NewDecoder(res.Body)

		didOnce := false

		for {
			res2 := new(StreamResponse)
			if err := dec.Decode(res2); err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			if !didOnce {
				didOnce = true
			}
			stepGoUp := res2.Step > step
			step = res2.Step
			if res2.TotalSteps != 0 {
				totalSteps = res2.TotalSteps
			}
			if res2.Output != nil {
				if stillTyping {
					stillTyping = false
					stoptyping <- struct{}{}
				}
				if config.StreamImageProgress && res2.Output[0].Path != "" {
					res3, err := Get(config.StableDiffusionURL + res2.Output[0].Path)
					if err != nil {
						s.EditMessage(c.ChannelID, msg.ID, "Failed to parse progress image")
					} else {
						if lastFrame != nil {
							s.DeleteMessage(lastFrame.ChannelID, lastFrame.ID, "progress frame")
							lastFrame = nil
						}
						f, err := frame(c.ChannelID, msg.ID, res3.Body, step, totalSteps, body.InitImage != "")
						if err == nil {
							lastFrame = f
						}
						res3.Body.Close()
					}
				} else if res2.Output[0].Data != "" {
					if lastFrame != nil {
						s.DeleteMessage(lastFrame.ChannelID, lastFrame.ID, "progress frame")
						lastFrame = nil
					}
					b64body := base64.NewDecoder(base64.StdEncoding, strings.NewReader(res2.Output[0].Data[22:]))
					f, err := frame(c.ChannelID, msg.ID, b64body, totalSteps, totalSteps, body.InitImage != "")
					if err == nil {
						lastFrame = f
					}
					doneRendering = true
					break
				}
			} else if res2.Status != "" {
				if lastFrame != nil {
					s.DeleteMessage(lastFrame.ChannelID, lastFrame.ID, "progress frame")
					lastFrame = nil
				}
				frameEmbed(c.ChannelID, msg.ID, config.ErrorFrameUrl, 0, 0, body.InitImage != "")
				doneRendering = true
				break
			} else if config.CountFrameless && stepGoUp {
				if stillTyping {
					stillTyping = false
					stoptyping <- struct{}{}
				}
				frame(c.ChannelID, msg.ID, nil, step, totalSteps, body.InitImage != "")
			}
		}

		res.Body.Close()

		if !didOnce {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func loadJson(path string, v interface{}) {
	data, err := os.ReadFile(path)

	if err != nil {
		log.Printf("Cannot load json: %v", err)
		return
	}

	err = json.Unmarshal(data, &v)
	if err != nil {
		log.Printf("Cannot unmarshal data: %v", err)
		return
	}
}

func main() {
	loadJson("config.json", &config)
	loadJson("users.json", &usersList)

	if config.BotToken == "" {
		log.Fatalln("Missing bot token!")
	}

	if len(config.ChannelIds) < 1 {
		log.Println("Missing channel IDs, will respond in all channels!")
	}

	if config.ImageDumpChannelId == "" {
		log.Println("Missing image dump channel ID, will dump images in the same channel!")
	} else {
		i, err := strconv.Atoi(config.ImageDumpChannelId)

		if err != nil {
			log.Fatalln("Invalid image dump channel ID!")
		}

		imageDumpChannelId = discord.ChannelID(discord.Snowflake(i))
	}

	babbler = babble.NewBabbler()
	babbler.Separator = ", "

	s = state.New("Bot " + config.BotToken)
	s.AddHandler(messageCreate)

	s.AddIntents(gateway.IntentGuildMessages)

	self, err := s.Me()
	if err != nil {
		log.Fatalln("Identity crisis:", err)
	}

	botID = self.ID

	ctx = context.Background()

	if err := s.Open(ctx); err != nil {
		log.Fatalln("Failed to connect:", err)
	}
	defer s.Close()

	log.Println("Started as", self.Username)

	if config.FrameUrl != "" {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			channel, channelInit := channels[strings.SplitN(r.RequestURI, "/", 3)[1]]
			if !channelInit || channel.FrameData == nil {
				w.Header().Add("Content-Type", "text/plain")
				w.WriteHeader(404)
				io.WriteString(w, "404 Not Found")
				return
			}
			w.Header().Add("Content-Type", "image/png")
			w.WriteHeader(200)
			w.Write(channel.FrameData)
		})

		err2 := http.ListenAndServe(config.FrameHttpBind, nil)

		if err2 != nil {
			log.Fatalln("Failed to start webserver:", err2)
		}
	}

	select {}
}
