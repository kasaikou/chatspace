package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	CredKey = "creds.json"
)

type cred struct {
	Env map[string]string `json:"environments"`
}

func LoadEnvCredentials() (map[string]string, error) {

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithDefaultRegion("ap-northeast-1"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	object, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(CredKey),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get object from s3: %w", err)
	}

	b, err := io.ReadAll(object.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read content body: %w", err)
	}

	c := cred{}
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("cannot parse content body: %w", err)
	}

	return c.Env, nil
}
