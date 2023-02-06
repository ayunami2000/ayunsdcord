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

var s *state.State
var botID discord.UserID
var ctx context.Context
var inUse sync.Mutex
var sessionId string = strconv.Itoa(int(time.Now().UnixMilli()))
var model string = "animemix"
var vae string = "anything"
var hypernetwork string = ""
var prompt string = "cat"
var negativePrompt string = "nsfw"
var width int = 768
var height int = 768
var babbler babble.Babbler

type Render struct {
	Prompt                  string   `json:"prompt"`
	Seed                    int      `json:"seed"`
	NegativePrompt          string   `json:"negative_prompt"`
	NumOutputs              int      `json:"num_outputs"`
	NumInferenceSteps       int      `json:"num_inference_steps"`
	GuidanceScale           int      `json:"guidance_scale"`
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

var TOKEN = os.Getenv("BOT_TOKEN")
var SD_URL = os.Getenv("SD_URL")
var BASIC_AUTH = os.Getenv("BASIC_AUTH")
var CHANNEL_ID = os.Getenv("CHANNEL_ID")
var IMAGE_DUMP_CHANNEL_ID_RAW = os.Getenv("IMAGE_DUMP_CHANNEL_ID")
var IMAGE_DUMP_CHANNEL_ID discord.ChannelID
var PREFIX = os.Getenv("PREFIX")

func Get(url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if BASIC_AUTH != "" {
		req.Header.Set("Authorization", "Basic "+BASIC_AUTH)
	}
	return http.DefaultClient.Do(req)
}

func Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if BASIC_AUTH != "" {
		req.Header.Set("Authorization", "Basic "+BASIC_AUTH)
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

func frame(channelId discord.ChannelID, referenceId discord.MessageID, reader io.Reader, step int, totalSteps int) (*discord.Message, error) {
	msg, err := s.SendMessageComplex(IMAGE_DUMP_CHANNEL_ID, api.SendMessageData{
		Files: []sendpart.File{{
			Name:   "stable-diffusion_" + strconv.Itoa(int(time.Now().UnixMilli())) + ".png",
			Reader: reader,
		}},
	})
	if err != nil {
		s.EditMessage(channelId, referenceId, "failed to upload progress image")
	} else {
		frameEmbed(channelId, referenceId, msg.Attachments[0].URL, step, totalSteps)
	}
	return msg, err
}

func frameEmbed(channelId discord.ChannelID, referenceId discord.MessageID, url string, step int, totalSteps int) {
	footer := "Done!"
	if step == 0 && totalSteps == 0 {
		footer = "Error."
	}
	if step != totalSteps {
		footer = "Step " + strconv.Itoa(step) + " of " + strconv.Itoa(totalSteps)
	}
	s.EditMessageComplex(channelId, referenceId, api.EditMessageData{
		Content: option.NewNullableString(""),
		Embeds: &[]discord.Embed{{
			Title:       "Stable Diffusion",
			Description: "**Prompt:** " + prompt + "\n**Negative Prompt:** " + negativePrompt + "\n**Width:** " + strconv.Itoa(width) + "\n**Height:** " + strconv.Itoa(height) + "\n**Model:** " + model + "\n**VAE:** " + vae + "\n**HyperNetwork:** " + hypernetwork,
			Footer: &discord.EmbedFooter{
				Text: footer,
			},
			Image: &discord.EmbedImage{
				URL: url,
			},
			Timestamp: discord.NewTimestamp(time.Now()),
		}},
	})
}

func getModels() (*ModelsResponse, error) {
	res, err := Get(SD_URL + "/get/models")
	if err != nil {
		return nil, err
	} else {
		defer res.Body.Close()
		resParsed := new(ModelsResponse)
		json.NewDecoder(res.Body).Decode(resParsed)
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

func messageCreate(c *gateway.MessageCreateEvent) {
	if c.Author.ID == botID {
		return
	}

	if c.ChannelID.String() != CHANNEL_ID {
		return
	}

	prefix := PREFIX

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

	if !inUse.TryLock() {
		reply(c.ChannelID, c.ID, "**Error:** Already in use.")
		return
	}

	if err := s.Typing(c.ChannelID); err != nil {
		log.Println("could not start typing:", err)
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
				inUse.Unlock()
				return
			}
		}
	}()

	cmd := strings.ToLower(strings.Split(args, " ")[0])
	theRest := strings.TrimSpace(args[len(cmd):])

	if cmd != "render" && cmd != "r" && cmd != "randomrender" && cmd != "rr" {
		if cmd == "help" || cmd == "h" || cmd == "?" {
			reply(c.ChannelID, c.ID, "**Usage:** "+prefix+" (command) [args]\n**Commands:** listmodels, model, vae, hypernetwork, clear, render, size, random, randomrender, help")
		} else if cmd == "random" || cmd == "rand" {
			pr, err := randomPrompt(theRest)
			if err != nil {
				reply(c.ChannelID, c.ID, "**Error:** Invalid number!")
			} else {
				prompt = truncateText(pr, 512)
				reply(c.ChannelID, c.ID, "**Prompt randomly set to:** "+prompt)
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
				reply(c.ChannelID, c.ID, "**Error:** Please specify which parameter to clear: prompt, negativeprompt, model, vae, hypernetwork")
			} else if strings.EqualFold(theRest, "prompt") || strings.EqualFold(theRest, "p") {
				prompt = ""
				reply(c.ChannelID, c.ID, "**Cleared the prompt!**")
			} else if strings.EqualFold(theRest, "negativeprompt") || strings.EqualFold(theRest, "np") {
				negativePrompt = ""
				reply(c.ChannelID, c.ID, "**Cleared the negative prompt!**")
			} else if strings.EqualFold(theRest, "model") || strings.EqualFold(theRest, "m") {
				reply(c.ChannelID, c.ID, "**Error:** The Model cannot be cleared!")
			} else if strings.EqualFold(theRest, "vae") || strings.EqualFold(theRest, "v") {
				vae = ""
				reply(c.ChannelID, c.ID, "**Cleared the VAE!**")
			} else if strings.EqualFold(theRest, "hypernetwork") || strings.EqualFold(theRest, "hypnet") || strings.EqualFold(theRest, "hn") {
				hypernetwork = ""
				reply(c.ChannelID, c.ID, "**Cleared the HyperNetwork!**")
			} else {
				reply(c.ChannelID, c.ID, "**Error:** Invalid parameter to clear!")
			}
		} else if cmd == "model" || cmd == "m" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current Model:** "+model)
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
						model = theRest
						reply(c.ChannelID, c.ID, "**Model set to:** "+model)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid model!")
					}
				}
			}
		} else if cmd == "vae" || cmd == "v" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current VAE:** "+vae)
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
						vae = theRest
						reply(c.ChannelID, c.ID, "**VAE set to:** "+vae)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid VAE!")
					}
				}
			}
		} else if cmd == "hypernetwork" || cmd == "hypnet" || cmd == "hn" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current HyperNetwork:** "+hypernetwork)
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
						hypernetwork = theRest
						reply(c.ChannelID, c.ID, "**HyperNetwork set to:** "+hypernetwork)
					} else {
						reply(c.ChannelID, c.ID, "**Error:** Invalid HyperNetwork!")
					}
				}
			}
		} else if cmd == "prompt" || cmd == "p" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current prompt:** "+prompt)
			} else {
				prompt = truncateText(theRest, 512)
				reply(c.ChannelID, c.ID, "**Prompt set to:** "+prompt)
			}
		} else if cmd == "negativeprompt" || cmd == "np" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current negative prompt:** "+negativePrompt)
			} else {
				negativePrompt = truncateText(theRest, 512)
				reply(c.ChannelID, c.ID, "**Negative prompt set to:** "+negativePrompt)
			}
		} else if cmd == "size" || cmd == "sz" {
			if theRest == "" {
				reply(c.ChannelID, c.ID, "**Current size:** "+strconv.Itoa(width)+"x"+strconv.Itoa(height)+"\n**Sizes:** 0: 768x768, 1: 1280x768, 2: 768x1280, 3: 512x512, 4: 896x512, 5: 512x896")
			} else {
				if theRest == "0" {
					width = 768
					height = 768
				} else if theRest == "1" {
					width = 1280
					height = 768
				} else if theRest == "2" {
					width = 768
					height = 1280
				} else if theRest == "3" {
					width = 512
					height = 512
				} else if theRest == "4" {
					width = 896
					height = 512
				} else if theRest == "5" {
					width = 512
					height = 896
				} else {
					reply(c.ChannelID, c.ID, "**Error:** Invalid size!")
					return
				}
				reply(c.ChannelID, c.ID, "**Size set to:** "+strconv.Itoa(width)+"x"+strconv.Itoa(height))
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
			prompt = truncateText(pr, 512)
			reply(c.ChannelID, c.ID, "**Prompt randomly set to:** "+prompt)
		}
	} else if theRest != "" {
		prompt = truncateText(theRest, 512)
		reply(c.ChannelID, c.ID, "**Prompt set to:** "+prompt)
	}

	body := &Render{
		Prompt:                  prompt,
		Seed:                    rand.Intn(1000000),
		NegativePrompt:          negativePrompt,
		NumOutputs:              1,
		NumInferenceSteps:       28,
		GuidanceScale:           12,
		Width:                   width,
		Height:                  height,
		VramUsageLevel:          "high",
		UseStableDiffusionModel: model,
		UseVaeModel:             vae,
		UseHypernetworkModel:    hypernetwork,
		StreamProgressUpdates:   true,
		StreamImageProgress:     true,
		ShowOnlyFilteredImage:   true,
		OutputFormat:            "png",
		OutputQuality:           75,
		MetadataOutputFormat:    "txt",
		OriginalPrompt:          prompt,
		ActiveTags:              []string{},
		InactiveTags:            []string{},
		SamplerName:             "euler_a",
		SessionId:               sessionId,
	}

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(body)
	res, err := Post(SD_URL+"/render", "application/json", buf)

	if err != nil {
		log.Println("could not query stable diffusion ui:", err)
		reply(c.ChannelID, c.ID, "failed to query stable diffusion ui")
		return
	}

	defer res.Body.Close()

	resParsed := new(RenderResponse)

	json.NewDecoder(res.Body).Decode(resParsed)

	msg, err := reply(c.ChannelID, c.ID, "**Loading...**")

	if err != nil {
		return
	}

	doneRendering := bool(false)

	lastFrame := new(discord.Message)

	step := 0
	totalSteps := 28

	for !doneRendering {
		res, err = Get(SD_URL + resParsed.Stream)

		if err != nil {
			log.Println("could not query stable diffusion ui:", err)
			s.EditMessage(c.ChannelID, msg.ID, "failed to query stable diffusion ui")
			return
		}

		defer res.Body.Close()

		dec := json.NewDecoder(res.Body)

		for {
			res2 := new(StreamResponse)
			if err := dec.Decode(res2); err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			if res2.Output != nil {
				if res2.Output[0].Path != "" {
					res3, err := Get(SD_URL + res2.Output[0].Path)
					if err != nil {
						s.EditMessage(c.ChannelID, msg.ID, "failed to parse progress image")
					} else {
						defer res3.Body.Close()
						if lastFrame != nil {
							s.DeleteMessage(lastFrame.ChannelID, lastFrame.ID, "progress frame")
							lastFrame = nil
						}
						f, err := frame(c.ChannelID, msg.ID, res3.Body, step, totalSteps)
						if err == nil {
							lastFrame = f
						}
					}
				} else if res2.Output[0].Data != "" {
					if lastFrame != nil {
						s.DeleteMessage(lastFrame.ChannelID, lastFrame.ID, "progress frame")
						lastFrame = nil
					}
					b64body := base64.NewDecoder(base64.StdEncoding, strings.NewReader(res2.Output[0].Data[22:]))
					f, err := frame(c.ChannelID, msg.ID, b64body, totalSteps, totalSteps)
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
				frameEmbed(c.ChannelID, msg.ID, "https://upload.wikimedia.org/wikipedia/commons/f/f7/Generic_error_message.png", 0, 0)
				doneRendering = true
				break
			}
			if res2.Step != 0 {
				step = res2.Step
			}
			if res2.TotalSteps != 0 {
				totalSteps = res2.TotalSteps
			}
		}
	}
}

func main() {
	if TOKEN == "" {
		log.Fatalln("missing BOT_TOKEN")
	}

	if CHANNEL_ID == "" {
		log.Fatalln("missing CHANNEL_ID")
	}

	if IMAGE_DUMP_CHANNEL_ID_RAW == "" {
		log.Fatalln("missing IMAGE_DUMP_CHANNEL_ID")
	}

	i, err := strconv.Atoi(IMAGE_DUMP_CHANNEL_ID_RAW)

	if err != nil {
		log.Fatalln("Invalid IMAGE_DUMP_CHANNEL_ID")
	}

	IMAGE_DUMP_CHANNEL_ID = discord.ChannelID(discord.Snowflake(i))

	if SD_URL == "" {
		SD_URL = "http://localhost:9000"
	}

	if PREFIX == "" {
		PREFIX = "sd!"
	}

	babbler = babble.NewBabbler()
	babbler.Separator = ", "

	res, err := Get(SD_URL + "/get/app_config")
	if err != nil {
		log.Fatalln("Failed to get active models!")
	} else {
		defer res.Body.Close()
		resParsed := new(AppConfigResponse)
		json.NewDecoder(res.Body).Decode(resParsed)
		model = resParsed.Model.StableDiffusion
		vae = resParsed.Model.VAE
		hypernetwork = resParsed.Model.HyperNetwork
	}

	s = state.New("Bot " + TOKEN)
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

	select {}
}
