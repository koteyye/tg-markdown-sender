package objectstore

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	testBucketName = "post-images"
	testPublicURL  = "https://storage.yandexcloud.net/post-images"
)

type fakeClient struct {
	putInput  *s3.PutObjectInput
	putErr    error
	headInput *s3.HeadBucketInput
	headErr   error
}

func (f *fakeClient) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putInput = input
	if f.putErr != nil {
		return nil, f.putErr
	}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeClient) HeadBucket(_ context.Context, input *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	f.headInput = input
	if f.headErr != nil {
		return nil, f.headErr
	}
	return &s3.HeadBucketOutput{}, nil
}

func TestStoreUploadPhoto(t *testing.T) {
	t.Parallel()

	storageClient := &fakeClient{}
	store := &Store{
		bucket:        testBucketName,
		publicBaseURL: testPublicURL,
		client:        storageClient,
	}

	publicURL, err := store.UploadPhoto(context.Background(), []byte("image-data"))
	if err != nil {
		t.Fatalf("UploadPhoto returned error: %v", err)
	}
	if !strings.HasPrefix(publicURL, "https://storage.yandexcloud.net/post-images/images/") || !strings.HasSuffix(publicURL, ".jpg") {
		t.Fatalf("unexpected public URL: %q", publicURL)
	}
	if aws.ToString(storageClient.putInput.Bucket) != testBucketName {
		t.Fatalf("unexpected bucket: %q", aws.ToString(storageClient.putInput.Bucket))
	}
	if aws.ToString(storageClient.putInput.ContentType) != "image/jpeg" {
		t.Fatalf("unexpected content type: %q", aws.ToString(storageClient.putInput.ContentType))
	}
	data, err := io.ReadAll(storageClient.putInput.Body)
	if err != nil {
		t.Fatalf("read upload data: %v", err)
	}
	if string(data) != "image-data" {
		t.Fatalf("unexpected upload data: %q", data)
	}
}

func TestStoreUploadPhotoFailure(t *testing.T) {
	t.Parallel()

	store := &Store{
		bucket:        testBucketName,
		publicBaseURL: testPublicURL,
		client:        &fakeClient{putErr: errors.New("upload failed")},
	}

	_, err := store.UploadPhoto(context.Background(), []byte("image-data"))
	if err == nil || !strings.Contains(err.Error(), "upload object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStoreCheck(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		storageClient := &fakeClient{}
		store := &Store{bucket: testBucketName, client: storageClient}

		if err := store.Check(context.Background()); err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if aws.ToString(storageClient.headInput.Bucket) != testBucketName {
			t.Fatalf("unexpected bucket: %q", aws.ToString(storageClient.headInput.Bucket))
		}
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()

		store := &Store{bucket: testBucketName, client: &fakeClient{headErr: errors.New("access denied")}}
		err := store.Check(context.Background())
		if err == nil || !strings.Contains(err.Error(), "check bucket access") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	validConfig := Config{
		Endpoint:      "https://storage.yandexcloud.net",
		Region:        "ru-central1",
		AccessKeyID:   "access-key",
		SecretKey:     "secret-key",
		Bucket:        testBucketName,
		PublicBaseURL: testPublicURL,
	}
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "valid configuration", config: validConfig},
		{
			name:    "HTTP endpoint is rejected",
			config:  Config{Endpoint: "http://storage.example.com", Region: validConfig.Region, AccessKeyID: validConfig.AccessKeyID, SecretKey: validConfig.SecretKey, Bucket: validConfig.Bucket, PublicBaseURL: validConfig.PublicBaseURL},
			wantErr: true,
		},
		{
			name:    "endpoint path is rejected",
			config:  Config{Endpoint: "https://storage.example.com/api", Region: validConfig.Region, AccessKeyID: validConfig.AccessKeyID, SecretKey: validConfig.SecretKey, Bucket: validConfig.Bucket, PublicBaseURL: validConfig.PublicBaseURL},
			wantErr: true,
		},
		{
			name:    "HTTP public URL is rejected",
			config:  Config{Endpoint: validConfig.Endpoint, Region: validConfig.Region, AccessKeyID: validConfig.AccessKeyID, SecretKey: validConfig.SecretKey, Bucket: validConfig.Bucket, PublicBaseURL: "http://media.example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
