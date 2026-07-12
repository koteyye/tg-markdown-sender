// Package objectstore uploads images to a public S3-compatible object store.
package objectstore

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config contains the connection and public URL settings for one S3-compatible bucket.
type Config struct {
	Endpoint      string
	Region        string
	AccessKeyID   string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
}

type client interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// Store uploads Telegram photos and returns their permanent public URLs.
type Store struct {
	bucket        string
	publicBaseURL string
	client        client
}

// New creates a store for an S3-compatible endpoint. It uses path-style requests,
// which are supported by Yandex Object Storage and Cloudflare R2.
func New(cfg Config) (*Store, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	awsConfig := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretKey, ""),
		Region:      cfg.Region,
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})

	return &Store{
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		client:        client,
	}, nil
}

// UploadPhoto stores an image as a JPEG and returns its permanent public URL.
func (s *Store) UploadPhoto(ctx context.Context, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("photo data is empty")
	}

	key, err := newObjectKey()
	if err != nil {
		return "", fmt.Errorf("create object key: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return "", fmt.Errorf("upload object: %w", err)
	}

	publicURL, err := url.JoinPath(s.publicBaseURL, key)
	if err != nil {
		return "", fmt.Errorf("build public image URL: %w", err)
	}
	return publicURL, nil
}

// Check verifies that the configured credentials can access the bucket.
func (s *Store) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return fmt.Errorf("check bucket access: %w", err)
	}
	return nil
}

func validateConfig(cfg Config) error {
	if err := validateRequiredFields(cfg); err != nil {
		return err
	}
	if err := validateHTTPSURL(cfg.Endpoint, "endpoint", false); err != nil {
		return err
	}
	return validateHTTPSURL(cfg.PublicBaseURL, "public base URL", true)
}

func validateRequiredFields(cfg Config) error {
	values := []struct {
		name  string
		value string
	}{
		{name: "endpoint", value: cfg.Endpoint},
		{name: "region", value: cfg.Region},
		{name: "access key ID", value: cfg.AccessKeyID},
		{name: "secret access key", value: cfg.SecretKey},
		{name: "bucket", value: cfg.Bucket},
		{name: "public base URL", value: cfg.PublicBaseURL},
	}
	for _, item := range values {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("object storage %s is required", item.name)
		}
	}
	return nil
}

func validateHTTPSURL(rawURL, name string, allowPath bool) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return fmt.Errorf("object storage %s must be an HTTPS URL without credentials, query, or fragment", name)
	}
	if !allowPath && parsedURL.Path != "" && parsedURL.Path != "/" {
		return fmt.Errorf("object storage %s must be an HTTPS URL without credentials, path, query, or fragment", name)
	}
	return nil
}

func newObjectKey() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "images/" + hex.EncodeToString(random[:]) + ".jpg", nil
}
