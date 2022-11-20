package registry

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

type S3Client struct {
	s3Client *s3.S3
	bucket   string
}

type Registry map[Variant]map[VariantVersion]Entry

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

func (c *S3Client) GetRegistry() (Registry, error) {
	file, err := c.GetObject(".registry")
	var latest Registry
	err = yaml.Unmarshal(file, &latest)
	if err != nil {
		return nil, err
	}
	return latest, nil
}

func (c *S3Client) GetObject(key string) ([]byte, error) {
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}
	// get file
	resp, err := c.s3Client.GetObject(getObjectInput)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
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

func (r Registry) String() string {
	builder := strings.Builder{}
	fmt.Fprintf(&builder, "\n")
	for variantName, variant := range r {
		for versionName, version := range variant {
			fmt.Fprintf(&builder, "------------ Scroll: %s@%s ------------\n", variantName, versionName)
			fmt.Fprintf(&builder, "Latest: %s\n", version.Latest.String())
		}
	}
	return builder.String()
}
