package sdapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ayunami2000/ayunsdcord/config"
)

var ErrResponseCode = errors.New("got unexpected response code")
var httpClient = http.Client{
	Timeout:   10 * time.Minute, // Hopefully this doesn't break models that take too long to load
	Transport: &httpTransport{},
}

type httpTransport struct{}

func getSDUrl() string {
	config.ConfigMutex.Lock()
	stableDiffusionURL := config.Config.StableDiffusionURL
	config.ConfigMutex.Unlock()

	return stableDiffusionURL
}

func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	config.ConfigMutex.Lock()
	if config.Config.BasicAuth != "" {
		req.Header.Set("Authorization", "Basic "+config.Config.BasicAuth)
	}
	config.ConfigMutex.Unlock()

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 && res.StatusCode != 425 {
		return res, fmt.Errorf("%w: %s", ErrResponseCode, res.Status)
	}

	return res, nil
}

func GetModels() (*ModelsResponse, error) {
	res, err := httpClient.Get(getSDUrl() + "/get/models")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var resParsed ModelsResponse
	if err := json.NewDecoder(res.Body).Decode(&resParsed); err != nil {
		return nil, err
	}

	return &resParsed, nil
}

func GetAppConfig() (*AppConfigResponse, error) {
	res, err := httpClient.Get(getSDUrl() + "/get/app_config")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	var resParsed AppConfigResponse

	err = json.NewDecoder(res.Body).Decode(&resParsed)
	return &resParsed, err
}

func Render(data *RenderData) (string, int64, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", 0, err
	}

	res, err := httpClient.Post(getSDUrl()+"/render", "application/json", &buf)
	if err != nil {
		return "", 0, err
	}

	defer res.Body.Close()

	var resParsed renderResponse
	err = json.NewDecoder(res.Body).Decode(&resParsed)
	return resParsed.Stream, resParsed.Task, err
}

func StopRender(task int64) error {
	res, err := httpClient.Get(getSDUrl() + "/image/stop?task=" + strconv.FormatInt(task, 10))
	if err != nil {
		return err
	}

	res.Body.Close()
	return nil
}

func GetStream(streamURL string) ([]StreamResponse, error) {
	res, err := httpClient.Get(getSDUrl() + streamURL)
	if err != nil {
		if res.StatusCode == 425 {
			return []StreamResponse{}, nil
		}
		return nil, err
	}

	responses := []StreamResponse{}
	var response StreamResponse
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()

	for {
		if err := decoder.Decode(&response); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return responses, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func GetImage(path string) (io.ReadCloser, error) {
	res, err := httpClient.Get(getSDUrl() + path)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}
