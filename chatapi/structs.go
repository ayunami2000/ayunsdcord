package chatapi

type KoboldRequest struct {
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

type KoboldResponse struct {
	Results []struct {
		Text string `json:"text"`
	} `json:"results"`
}

type TogetherResponse struct {
	Output struct {
		Choices []struct {
			Text string `json:"text"`
		}
	}
}

type OpenAIRequest struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        uint    `json:"max_tokens"`
	TopP             float64 `json:"top_p"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	PresencePenalty  float64 `json:"presence_penalty"`
	User             string  `json:"user,omitempty"`
}

type OpenAIResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

type KoboldHordeRequestParams struct {
	N                uint    `json:"n"`
	MaxContextLength uint    `json:"max_context_length"`
	MaxLength        uint    `json:"max_length"`
	RepPen           float64 `json:"rep_pen"`
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
}

type KoboldHordeRequest struct {
	Prompt         string                   `json:"prompt"`
	Params         KoboldHordeRequestParams `json:"params"`
	TrustedWorkers bool                     `json:"trusted_workers"`
	NSFW           bool                     `json:"nsfw"`
}

type KoboldHordeInitialResponse struct {
	ID string `json:"id"`
}

type KoboldHordeStatusResponse struct {
	IsPossible  bool `json:"is_possible"`
	Done        bool `json:"done"`
	Generations []struct {
		Text string `json:"text"`
	} `json:"generations,omitempty"`
}
