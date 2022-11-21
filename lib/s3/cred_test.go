package s3_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	target "github.com/streamwest-1629/chatspace/lib/s3"
	"go.uber.org/zap"
)

func TestSaveCreds(t *testing.T) {

	logger, _ := zap.NewDevelopment()
	if err := saveCredentials(); err != nil {
		logger.Fatal("failed saving as s3 object", zap.Error(err))
	}

	env, err := target.LoadEnvCredentials()
	if err != nil {
		logger.Fatal("failed loading from s3 object", zap.Error(err))
	}

	fields := []zap.Field{}
	for key, value := range env {
		fields = append(fields, zap.String(key, value))
	}

	logger.Info("loaded environment credentials", fields...)
}

func saveCredentials() error {
	file, err := os.Open(os.Getenv("RELEASE_CRED_JSON"))
	if err != nil {
		return fmt.Errorf("cannot open local file: %w", err)
	}
	defer file.Close()

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithDefaultRegion("ap-northeast-1"),
	)
	if err != nil {
		return fmt.Errorf("cannot load aws configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(target.CredKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("cannot save credentials: %w", err)
	}

	return nil
}
