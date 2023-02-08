# ayunsdcord
stable diffusion ui discord bot

built for use with only one image at a time

### getting started:
minimum:
```batch
@echo off
set "BOT_TOKEN="
set "CHANNEL_IDS=0000000000000000000"
set "IMAGE_DUMP_CHANNEL_ID=0000000000000000000"
go run .
```

config values & environment variables:
- click [here](https://regex101.com/r/AMGrCX/1)
- paste in the contents of `main.go`
- instantly get a rundown of:
  - the config,
  - the environment variables, and
  - their default values

base config.json:
```json
{
	"bot_token": "",
	"channel_ids": [
		"0000000000000000000"
	],
	"image_dump_channel_id": "0000000000000000000"
}
```

sample config.json:
```json
{
	"bot_token": "",
	"channel_ids": [
		"0000000000000000000",
		"0000000000000000000"
	],
	"image_dump_channel_id": "0000000000000000000",
	"prefix": "@sd",
	"allow_bots": true,
	"default_prompt": "cat",
	"default_width": 512,
	"default_height": 512,
	"default_inference_steps": 28,
	"default_guidance_scale": 12,
	"default_negative_prompt": "nsfw, child, children, loli",
	"default_upscaler": "RealESRGAN_x4plus",
	"default_upscale_amount": "2",
	"deny_changing": [
		"size",
		"upscale_amount",
		"inference_steps",
		"guidance_scale"
	],
	"count_frameless": false,
	"frame_url": "https://sd-bot.my-cool-web.site",
	"frame_http_bind": ":8080"
}
```