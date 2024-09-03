package services

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Service struct {
	client         *minio.Client
	mp3BucketName  string
	webpBucketName string
	publicEndpoint string
}

func NewS3Service() (*S3Service, error) {
	publicEndpoint, ok := os.LookupEnv("S3_PUBLIC_ENDPOINT")
	if !ok {
		return nil, errors.New("S3_PUBLIC_ENDPOINT environment variable not set")
	}

	privateEndpoint, ok := os.LookupEnv("S3_PRIVATE_ENDPOINT")
	if !ok {
		return nil, errors.New("S3_PRIVATE_ENDPOINT environment variable not set")
	}

	accessKeyID, ok := os.LookupEnv("S3_ACCESS_KEY_ID")
	if !ok {
		return nil, errors.New("S3_ACCESS_KEY_ID environment variable not set")
	}

	secretAccessKey, ok := os.LookupEnv("S3_SECRET_ACCESS_KEY")
	if !ok {
		return nil, errors.New("S3_SECRET_ACCESS_KEY environment variable not set")
	}

	client, err := minio.New(privateEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	mp3BucketName, ok := os.LookupEnv("S3_MP3_BUCKET")
	if !ok {
		return nil, errors.New("S3_MP3_BUCKET environment variable not set")
	}

	webpBucketName, ok := os.LookupEnv("S3_WEBP_BUCKET")
	if !ok {
		return nil, errors.New("S3_WEBP_BUCKET environment variable not set")
	}

	return &S3Service{
		client:         client,
		mp3BucketName:  mp3BucketName,
		webpBucketName: webpBucketName,
		publicEndpoint: publicEndpoint,
	}, nil
}

func (s3Service S3Service) UploadMP3(ctx context.Context, filePath string, objectName string) (string, error) {
	info, err := s3Service.client.FPutObject(ctx, s3Service.mp3BucketName, objectName, filePath, minio.PutObjectOptions{ContentType: "audio/mpeg"})
	if err != nil {
		return "", fmt.Errorf("failed to put MP3 object: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s3Service.publicEndpoint, s3Service.mp3BucketName, info.Key), nil
}

func (s3Service S3Service) UploadWebP(ctx context.Context, filePath string, objectName string) (string, error) {
	info, err := s3Service.client.FPutObject(ctx, s3Service.webpBucketName, objectName, filePath, minio.PutObjectOptions{ContentType: "image/webp"})
	if err != nil {
		return "", fmt.Errorf("failed to put WebP object: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s3Service.publicEndpoint, s3Service.webpBucketName, info.Key), nil
}
