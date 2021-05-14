package cloud

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultMetadataURL = "http://169.254.169.254"
	defaultTimeout     = 30 * time.Second
)

type Metadata struct {
	apiURL *url.URL
	client *http.Client
}

func NewMetadata() (*Metadata, error) {
	apiURL, err := url.ParseRequestURI(defaultMetadataURL)
	if err != nil {
		return nil, err
	}
	return &Metadata{
		apiURL: apiURL,
		client: &http.Client{Timeout: defaultTimeout},
	}, nil
}

func (m *Metadata) createGetRequest(path string) (*http.Request, error) {
	u, err := m.apiURL.Parse(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (m *Metadata) do(req *http.Request) (string, error) {
	resp, err := m.client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (m *Metadata) CloudServerID() (string, error) {
	request, err := m.createGetRequest("/latest/meta-data/instance-id")
	if err != nil {
		return "", err
	}
	id, err := m.do(request)
	if err != nil {
		return "", err
	}
	return id, nil
}
