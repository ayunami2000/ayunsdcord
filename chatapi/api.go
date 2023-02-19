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
var ErrEmptyResponse = errors.New("no text in response")
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
		} else if strings.EqualFold(config.Config.ChatAPIMode, "koboldhorde") {
			req.Header.Set("apikey", config.Config.ChatAuth)
		} else {
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

func getChatUrl() string {
	config.ConfigMutex.Lock()
	chatURL := config.Config.ChatURL
	config.ConfigMutex.Unlock()

	return chatURL
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
			Params: KoboldHordeRequestParams{
				N:                1,
				MaxContextLength: 1024,
				MaxLength:        256,
				RepPen:           1.0,
				Temperature:      0.7,
				TopP:             1.0,
			},
			TrustedWorkers: false,
			NSFW:           false,
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

	res, err := httpClient.Post(getChatUrl(), "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed KoboldResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil {
		return "", err
	}

	if len(resParsed.Results) < 1 {
		return "", ErrEmptyResponse
	}

	return resParsed.Results[0].Text, err
}

func GenerateTogether(prompt string) (string, error) {
	res, err := httpClient.Get(getChatUrl() + "?model=Together-gpt-JT-6B-v1&prompt=" + url.QueryEscape(prompt) + "&top_p=1.0&top_k=40&temperature=1.0&max_tokens=256&repetition_penalty=1.0&stop=")
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed TogetherResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil {
		return "", err
	}

	if len(resParsed.Output.Choices) < 1 {
		return "", ErrEmptyResponse
	}

	return resParsed.Output.Choices[0].Text, err
}

func GenerateOpenAI(data *OpenAIRequest) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	res, err := httpClient.Post(getChatUrl(), "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resParsed OpenAIResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)

	if err != nil {
		return "", err
	}

	if len(resParsed.Choices) < 1 {
		return "", ErrEmptyResponse
	}

	return resParsed.Choices[0].Text, err
}

func GenerateSimple(prompt string) (string, error) {
	res, err := httpClient.Get(getChatUrl() + url.QueryEscape(prompt))
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
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	res, err := httpClient.Post(getChatUrl()+"/v2/generate/async", "application/json", &buf)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var resInitParsed KoboldHordeInitialResponse
	err = json.NewDecoder(res.Body).Decode(&resInitParsed)

	if err != nil {
		return "", err
	}

	reqID := resInitParsed.ID

	isDone := false

	for {
		action := "check"
		if isDone {
			action = "status"
		}
		res, err = httpClient.Get(getChatUrl() + "/v2/generate/" + action + "/" + reqID)
		if err != nil {
			return "", err
		}

		defer res.Body.Close()

		var resParsed KoboldHordeStatusResponse
		err = json.NewDecoder(res.Body).Decode(&resParsed)

		if err != nil {
			return "", err
		}

		if isDone {
			if len(resParsed.Generations) < 1 {
				return "", ErrEmptyResponse
			}

			return resParsed.Generations[0].Text, err
		}

		if !resParsed.IsPossible {
			req, err := http.NewRequest("DELETE", getChatUrl()+"/v2/generate/status/"+reqID, nil)
			if err != nil {
				return "", err
			}
			res, err = httpClient.Do(req)
			if err != nil {
				return "", err
			}
			defer res.Body.Close()
			return "", ErrEmptyResponse
		}

		if resParsed.Done {
			isDone = true
		}

		time.Sleep(500 * time.Millisecond)
	}
}
