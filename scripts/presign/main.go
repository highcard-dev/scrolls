package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {

	//grep credentials from env
	s3Endpoint := os.Getenv("PRESIGN_S3_ENDPOINT")
	bucketName := os.Getenv("PRESIGN_BUCKET_NAME")
	objectKey := os.Getenv("PRESIGN_OBJECT_KEY")
	accessKey := os.Getenv("PRESIGN_ACCESS_KEY")
	secretKey := os.Getenv("PRESIGN_SECRET_KEY")
	region := os.Getenv("PRESIGN_REGION")
	if region == "" {
		region = "us-east-1"
	}
	expiration := 3600 * 4

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           s3Endpoint,
				SigningRegion: region,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Create the PUT presigned URL
	presigner := s3.NewPresignClient(client)
	presignedRequest, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(time.Duration(expiration)*time.Second))
	if err != nil {
		log.Fatalf("failed to generate presigned URL: %v", err)
	}

	fmt.Println(presignedRequest.URL)
}
