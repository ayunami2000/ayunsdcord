package sdapi

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

type RenderData struct {
	Prompt                  string   `json:"prompt"`
	Seed                    int      `json:"seed"`
	NegativePrompt          string   `json:"negative_prompt"`
	NumOutputs              uint     `json:"num_outputs"`
	NumInferenceSteps       uint     `json:"num_inference_steps"`
	GuidanceScale           float64  `json:"guidance_scale"`
	Width                   uint     `json:"width"`
	Height                  uint     `json:"height"`
	VramUsageLevel          string   `json:"vram_usage_level"`
	UseStableDiffusionModel string   `json:"use_stable_diffusion_model"`
	UseVaeModel             string   `json:"use_vae_model,omitempty"`
	UseHypernetworkModel    string   `json:"use_hypernetwork_model,omitempty"`
	StreamProgressUpdates   bool     `json:"stream_progress_updates"`
	StreamImageProgress     bool     `json:"stream_image_progress"`
	ShowOnlyFilteredImage   bool     `json:"show_only_filtered_image"`
	OutputFormat            string   `json:"output_format"`
	OutputQuality           uint     `json:"output_quality"`
	MetadataOutputFormat    string   `json:"metadata_output_format"`
	OriginalPrompt          string   `json:"original_prompt"`
	ActiveTags              []string `json:"active_tags"`
	InactiveTags            []string `json:"inactive_tags"`
	SamplerName             string   `json:"sampler_name"`
	SessionId               string   `json:"session_id"`
	InitImage               string   `json:"init_image,omitempty"`
	PromptStrength          float64  `json:"prompt_strength,omitempty"`
	UseUpscale              string   `json:"use_upscale,omitempty"`
	UpscaleAmount           string   `json:"upscale_amount,omitempty"`
}

type renderResponse struct {
	Stream string `json:"stream"`
	Task   int64  `json:"task"`
}

type StreamResponse struct {
	Output []struct {
		Path string `json:"path,omitempty"`
		Data string `json:"data,omitempty"`
	} `json:"output"`
	Step       uint   `json:"step,omitempty"`
	TotalSteps uint   `json:"total_steps,omitempty"`
	Status     string `json:"status,omitempty"`
}
