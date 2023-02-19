package chatapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ayunami2000/ayunsdcord/config"
)

var ErrResponseCode = errors.New("got unexpected response code")
var httpClient = http.Client{
	Timeout:   10 * time.Minute,
	Transport: &httpTransport{},
}

type httpTransport struct{}

func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	config.ConfigMutex.Lock()
	if config.Config.ChatAuth != "" {
		if strings.EqualFold(config.Config.ChatAPIMode, "openai") {
			req.Header.Set("Authorization", "Bearer "+config.Config.ChatAuth)
		} else if !strings.EqualFold(config.Config.ChatAPIMode, "koboldhorde") {
			req.Header.Set("Authorization", "Basic "+config.Config.ChatAuth)
		}
	}
	config.ConfigMutex.Unlock()

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 && res.StatusCode != 425 {
		return nil, fmt.Errorf("%w: %s", ErrResponseCode, res.Status)
	}

	return res, nil
}

func Generate(prompt string) (string, error) {
	config.ConfigMutex.Lock()
	chatMode := config.Config.ChatAPIMode
	config.ConfigMutex.Unlock()

	if strings.EqualFold(chatMode, "kobold") {
		return GenerateKobold(&KoboldRequest{
			Prompt:      prompt,
			Temperature: 0.7,
			TopP:        1.0,
		})
	} else if strings.EqualFold(chatMode, "together") {
		return GenerateTogether(prompt)
	} else if strings.EqualFold(chatMode, "openai") {
		return GenerateOpenAI(&OpenAIRequest{
			Model:            "text-davinci-003",
			Prompt:           prompt,
			MaxTokens:        256,
			Temperature:      0.7,
			TopP:             1.0,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
			User:             "https://github.com/ayunami2000/ayunsdcord",
		})
	} else if strings.EqualFold(chatMode, "koboldhorde") {
		return GenerateKoboldHorde(&KoboldHordeRequest{
			Prompt: prompt,
			ApiKey: "0000000000",
			Params: KoboldHordeRequestParams{
				N:                1,
				MaxContextLength: 1024,
				MaxLength:        256,
				RepPen:           1.0,
				Temperature:      0.7,
				TopP:             1.0,
			},
			Servers: []string{},
			Models: []string{
				"facebook_opt-125m",
				"facebook/opt-13b",
			},
		})
	} else {
		return GenerateSimple(prompt)
	}
}

func GenerateKobold(data *KoboldRequest) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Post(chatURL, "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed KoboldResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil || len(resParsed.Results) < 1 {
		return "", err
	}

	return resParsed.Results[0].Text, err
}

func GenerateTogether(prompt string) (string, error) {
	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Get(chatURL + "?model=Together-gpt-JT-6B-v1&prompt=" + url.QueryEscape(prompt) + "&top_p=1.0&top_k=40&temperature=1.0&max_tokens=256&repetition_penalty=1.0&stop=")
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed TogetherResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil || len(resParsed.Output.Choices) < 1 {
		return "", err
	}

	return resParsed.Output.Choices[0].Text, err
}

func GenerateOpenAI(data *OpenAIRequest) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Post(chatURL, "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed OpenAIResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil || len(resParsed.Choices) < 1 {
		return "", err
	}

	return resParsed.Choices[0].Text, err
}

func GenerateSimple(prompt string) (string, error) {
	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Get(chatURL + url.QueryEscape(prompt))
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(b), err
}

func GenerateKoboldHorde(data *KoboldHordeRequest) (string, error) {
	config.ConfigMutex.Lock()
	if config.Config.ChatAuth != "" {
		data.ApiKey = config.Config.ChatAuth
	}
	config.ConfigMutex.Unlock()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Post(chatURL, "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed KoboldHordeResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil || len(resParsed) < 1 {
		return "", err
	}

	return resParsed[0].Text, err
}
