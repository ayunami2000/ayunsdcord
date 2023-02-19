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
	Prompt  string                   `json:"prompt"`
	ApiKey  string                   `json:"api_key"`
	Params  KoboldHordeRequestParams `json:"params"`
	Servers []string                 `json:"servers"`
	Models  []string                 `json:"models"`
}

type KoboldHordeResponse []struct {
	Text          string `json:"text"`
	ServerID      string `json:"server_id"`
	ServerName    string `json:"server_name"`
	QueuePosition uint   `json:"queue_position"`
}