package registry

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	BaseURL    *url.URL
	httpClient *http.Client
	bucket     string
}

type S3Client struct {
	s3Client *s3.S3
	bucket   string
}

type Registry map[Variant]map[VariantVersion]Entry

func NewClient(endpoint string, bucket string, apiKey string, apiSecret string) (*Client, error) {
	var user *url.Userinfo
	if apiKey != "" && apiSecret != "" {
		user = url.UserPassword(apiKey, apiSecret)
	}
	return &Client{BaseURL: &url.URL{Host: endpoint, Scheme: "https", User: user}, bucket: bucket, httpClient: http.DefaultClient}, nil
}

func NewS3Client(endpoint string, bucket string, apiKey string, apiSecret string) (*S3Client, error) {
	partsBase := strings.Split(endpoint, ".")
	if len(endpoint) < 2 {
		return nil, fmt.Errorf("invalid registry endpoint (base)")
	}
	s3Config := aws.Config{
		Credentials: credentials.NewStaticCredentials(apiKey, apiSecret, ""),
		//Endpoint:         aws.String("https://s3.wasabisys.com"),
		Endpoint:         aws.String(fmt.Sprintf("https://%s", endpoint)),
		Region:           aws.String(partsBase[1]),
		S3ForcePathStyle: aws.Bool(true),
	}
	goSession, err := session.NewSessionWithOptions(session.Options{
		Config: s3Config,
	})
	if err != nil {
		return nil, err
	}
	return &S3Client{s3Client: s3.New(goSession), bucket: bucket}, nil
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

func (c *S3Client) PutObject(path string, key string) error {
	//set the file path to upload
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	putObjectInput := &s3.PutObjectInput{
		Body:   file,
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	_, err = c.s3Client.PutObject(putObjectInput)

	if err != nil {
		return err
	}
	return nil
}
