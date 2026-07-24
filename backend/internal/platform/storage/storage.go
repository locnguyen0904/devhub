// Package storage presigns direct-to-S3 uploads.
//
// Files go straight from the browser to S3 using a presigned PUT URL — the API
// never holds the bytes. The design and the client contract are in
// docs/03-api.md §8.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config is the S3-compatible target. In dev this is MinIO; in production it can
// be R2 or AWS S3 by changing these values only.
type Config struct {
	Endpoint  string // e.g. http://localhost:9000 for MinIO
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	PublicURL string // base URL an uploaded object is served from
}

// Storage presigns uploads against one bucket.
type Storage struct {
	client    *s3.PresignClient
	bucket    string
	publicURL string
}

// New builds the presign client. UsePathStyle is required for MinIO: virtual
// host style (bucket.host) does not resolve against a bare localhost.
func New(cfg Config) *Storage {
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(cfg.Endpoint),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	})
	return &Storage{
		client:    s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}
}

// Presigned is a PUT URL the browser uploads to, plus the URL the object will be
// served from once uploaded.
type Presigned struct {
	UploadURL string
	PublicURL string
	ExpiresIn int // seconds
}

// PresignPut signs a PUT for the given object key and content type. The signature
// pins the content type, so the browser must send the same one it asked for.
func (s *Storage) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (Presigned, error) {
	req, err := s.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return Presigned{}, fmt.Errorf("presign put: %w", err)
	}
	return Presigned{
		UploadURL: req.URL,
		PublicURL: fmt.Sprintf("%s/%s", s.publicURL, key),
		ExpiresIn: int(ttl.Seconds()),
	}, nil
}
