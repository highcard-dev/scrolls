package registry

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL    *url.URL
	bucket     string
	httpClient *http.Client
}

type Registry map[Variant]map[VariantVersion]Entry

func NewClient(endpoint string, apiKey string, apiSecret string) (*Client, error) {
	var user *url.Userinfo
	parts := strings.Split(endpoint, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid registry endpoint")
	}
	if apiKey != "" && apiSecret != "" {
		user = url.UserPassword(apiKey, apiSecret)
	}
	return &Client{BaseURL: &url.URL{Host: parts[0], Scheme: "https", User: user}, bucket: parts[1], httpClient: http.DefaultClient}, nil
}

func (c *Client) GetRegistry() (Registry, error) {
	rel := &url.URL{Path: fmt.Sprintf("%s/.registry", c.bucket)}
	formattedUrl := c.BaseURL.ResolveReference(rel)
	req, err := http.NewRequest("GET", formattedUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/yaml")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error when retrieving .registry: status code %d - %s", response.StatusCode, response.Status)
	}
	defer response.Body.Close()
	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var latest Registry
	err = yaml.Unmarshal(b, &latest)
	if err != nil {
		return nil, err
	}
	return latest, nil
}
