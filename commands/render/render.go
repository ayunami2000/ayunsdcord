package render

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/ayunami2000/ayunsdcord/commands/command"
	"github.com/ayunami2000/ayunsdcord/config"
	"github.com/ayunami2000/ayunsdcord/sdapi"
	"github.com/ayunami2000/ayunsdcord/utils"
	"github.com/diamondburned/arikawa/v3/discord"
)

var ErrAlreadyInProgress = errors.New("render already in progress")
var RenderCommand = command.NewCommand("render", []string{"randomrender", "rr", "r"}, Run)

func Run(cmdctx *command.CommandContext) error {
	if !cmdctx.ChannelSettings.InUse.CompareAndSwap(false, true) {
		return ErrAlreadyInProgress
	}

	defer cmdctx.ChannelSettings.InUse.Store(false)

	if cmdctx.Args != "" && config.CanChange_NoLock("prompt") {
		cmdctx.ChannelSettings.Prompt = utils.TruncateText(cmdctx.Args, 512)
	}

	data := &sdapi.RenderData{
		Prompt:                  cmdctx.ChannelSettings.Prompt,
		Seed:                    int(rand.Int31()),
		NegativePrompt:          cmdctx.ChannelSettings.NegativePrompt,
		NumOutputs:              1,
		NumInferenceSteps:       cmdctx.ChannelSettings.InferenceSteps,
		GuidanceScale:           cmdctx.ChannelSettings.GuidanceScale,
		Width:                   cmdctx.ChannelSettings.Width,
		Height:                  cmdctx.ChannelSettings.Height,
		VramUsageLevel:          "high",
		UseStableDiffusionModel: cmdctx.ChannelSettings.Model,
		StreamProgressUpdates:   true,
		StreamImageProgress:     config.Config.StreamImageProgress,
		ShowOnlyFilteredImage:   true,
		OutputFormat:            "png",
		OutputQuality:           75,
		MetadataOutputFormat:    "txt",
		OriginalPrompt:          cmdctx.ChannelSettings.Prompt,
		ActiveTags:              []string{},
		InactiveTags:            []string{},
		SamplerName:             cmdctx.ChannelSettings.Sampler,
		SessionId:               cmdctx.ChannelSettings.SessionID,
		UseVaeModel:             cmdctx.ChannelSettings.VAE,
		UseHypernetworkModel:    cmdctx.ChannelSettings.HyperNetwork,
		UseUpscale:              cmdctx.ChannelSettings.Upscaler,
	}

	if cmdctx.ChannelSettings.Upscaler != "" {
		data.UpscaleAmount = strconv.FormatUint(uint64(cmdctx.ChannelSettings.UpscaleAmount), 10)
	}

	attachments := cmdctx.Message.Attachments
	hasImageAttachment := len(attachments) > 0 && strings.HasPrefix(attachments[0].ContentType, "image/")
	if hasImageAttachment {
		if err := config.CanChange(cmdctx.ChannelSettings.InUse, "img2img"); err != nil {
			attachment := attachments[0]
			if err := img2img(cmdctx, attachment, data); err != nil {
				_, err := cmdctx.TryReply("**Error:** Failed to download image for Img2Img!")
				if err != nil {
					return err
				}
			}
		} else {
			_, err := cmdctx.TryReply(fmt.Sprintf("**Error:** %v", err))
			if err != nil {
				return err
			}
		}
	}

	streamurl, task, err := sdapi.Render(data)
	if err != nil {
		log.Println("Could not query stable diffusion ui:", err)
		return err
	}

	msg, err := cmdctx.TryReply("**Loading...**")
	if err != nil {
		return err
	}

	var currentFrame *discord.Message
	currentStep := uint(0)
	config.ConfigMutex.Lock()
	totalSteps := cmdctx.ChannelSettings.InferenceSteps
	countFrameless := config.Config.CountFrameless
	errorFrameUrl := config.Config.ErrorFrameUrl
	stillTyping := true

	cmdctx.ChannelSettings.CurrentRenderInfoMutex.Lock()
	cmdctx.ChannelSettings.CurrentRenderInfo = &command.CurrentRenderInfo{
		RequestedBy:  cmdctx.Message.Author.ID,
		LastFrameUrl: config.Config.LoadingFrameUrl,
		Task:         task,
	}
	cmdctx.ChannelSettings.CurrentRenderInfoMutex.Unlock()
	config.ConfigMutex.Unlock()

	for currentStep < totalSteps {
		responses, err := sdapi.GetStream(streamurl)
		if err != nil {
			return err
		}

		var currentResponse *sdapi.StreamResponse
		for _, response := range responses {
			if len(response.Output) < 1 || (response.Output[0].Data == "" && response.Output[0].Path == "") {
				if response.Status != "" && response.Status != "succeeded" {
					_ = frameEmbed(cmdctx, msg, errorFrameUrl, 0, 0, data.InitImage != "")
					return fmt.Errorf("**Error:** Received error from stable diffusion: %s", response.Status)
				}

				if currentResponse.TotalSteps != 0 {
					totalSteps = currentResponse.TotalSteps
				}

				if response.Step > currentStep {
					currentStep = response.Step
					if countFrameless {
						_ = frameEmbed(cmdctx, msg, "", currentStep, totalSteps, data.InitImage != "")
					}
				}

				continue
			}

			if response.Status == "succeeded" {
				currentStep = totalSteps
				currentResponse = &response
				break
			}

			if response.Step <= currentStep {
				continue
			}

			currentStep = response.Step
			currentResponse = &response
		}

		if currentResponse == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if stillTyping {
			cmdctx.StopTyping <- struct{}{}
			stillTyping = false
		}

		if currentResponse.TotalSteps != 0 {
			totalSteps = currentResponse.TotalSteps
		}

		if currentResponse.Output[0].Data != "" {
			b64body := base64.NewDecoder(base64.StdEncoding, strings.NewReader(currentResponse.Output[0].Data[22:]))
			_, err := frame(cmdctx, msg, b64body, currentStep, totalSteps, data.InitImage != "")
			if currentFrame != nil {
				_ = cmdctx.Executor.DeleteMessage(currentFrame.ChannelID, currentFrame.ID, "progress frame")
			}

			if err != nil {
				_, _ = cmdctx.Executor.EditMessage(msg.ChannelID, msg.ID, fmt.Sprintf("**Error:** Failed to upload image: %v", err))
			}

			continue
		}

		image, err := sdapi.GetImage(currentResponse.Output[0].Path)
		if err != nil {
			_, _ = cmdctx.Executor.EditMessage(msg.ChannelID, msg.ID, fmt.Sprintf("**Error:** Failed to get image: %v", err))
			continue
		}

		f, err := frame(cmdctx, msg, image, currentStep, totalSteps, data.InitImage != "")
		if currentFrame != nil {
			_ = cmdctx.Executor.DeleteMessage(currentFrame.ChannelID, currentFrame.ID, "progress frame")
		}

		currentFrame = f

		if err != nil {
			_, _ = cmdctx.Executor.EditMessage(msg.ChannelID, msg.ID, fmt.Sprintf("**Error:** Failed to upload image: %v", err))
		}
	}

	return nil
}
