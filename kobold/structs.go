package kobold

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
