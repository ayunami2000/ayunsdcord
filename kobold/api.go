package kobold

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ayunami2000/ayunsdcord/config"
)

var ErrResponseCode = errors.New("got unexpected response code")
var httpClient = http.Client{
	Timeout:   10 * time.Minute, // Hopefully this doesn't break models that take too long to load
	Transport: &httpTransport{},
}

type httpTransport struct{}

func (t *httpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	config.ConfigMutex.Lock()
	if config.Config.KoboldBasicAuth != "" {
		req.Header.Set("Authorization", "Basic "+config.Config.KoboldBasicAuth)
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

func Generate(data *KoboldRequest) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return "", err
	}

	config.ConfigMutex.Lock()
	koboldURL := config.Config.KoboldURL
	config.ConfigMutex.Unlock()

	res, err := httpClient.Post(koboldURL, "application/json", &buf)
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
