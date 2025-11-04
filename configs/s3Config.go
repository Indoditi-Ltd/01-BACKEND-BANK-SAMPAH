package configs

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
}

var (
	S3Client *s3.Client
	s3Config S3Config
)

func InitS3() error {
	s3Config = S3Config{
		Region:          os.Getenv("AWS_REGION"),
		Bucket:          os.Getenv("AWS_BUCKET"),
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Endpoint:        os.Getenv("AWS_ENDPOINT"),
	}

	fmt.Printf("NEO Object Storage Config:\n")
	fmt.Printf("  Region: %s\n", s3Config.Region)
	fmt.Printf("  Bucket: %s\n", s3Config.Bucket)
	fmt.Printf("  Endpoint: %s\n", s3Config.Endpoint)
	fmt.Printf("  AccessKey: %s...\n", s3Config.AccessKeyID[:8])

	// Custom resolver untuk NEO Object Storage
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if s3Config.Endpoint != "" {
			return aws.Endpoint{
				URL:           s3Config.Endpoint,
				SigningRegion: s3Config.Region,
				Source:        aws.EndpointSourceCustom,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(s3Config.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyID,
			s3Config.SecretAccessKey,
			"",
		)),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client dengan Path Style untuk NEO Object Storage
	S3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if s3Config.Endpoint != "" {
			o.UsePathStyle = true // PAKAI PATH STYLE untuk NEO
		}
	})

	fmt.Printf("S3 client initialized successfully with Path Style\n")
	return nil
}

// Getter functions
func GetS3Bucket() string {
	return s3Config.Bucket
}

func GetAWSRegion() string {
	return s3Config.Region
}

func GetAWSEndpoint() string {
	return s3Config.Endpoint
}