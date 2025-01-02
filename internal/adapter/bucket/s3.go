package bucket

import (
	"context"
	"io"
	"bytes"
	"encoding/json"

	"github.com/go-order/internal/config/config_aws"
	"github.com/rs/zerolog/log"

	"github.com/go-order/internal/core"
	"github.com/go-order/internal/lib"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var childLogger = log.With().Str("bucket", "s3").Logger()

type BucketWorker struct {
	client 		*s3.Client
	bucketConfig core.BucketConfig
}

func NewBucketWorker(ctx context.Context, 
					bucketConfig core.BucketConfig) (*BucketWorker, error) {
	childLogger.Debug().Msg("NewClientS3Bucket")

	span := lib.Span(ctx, "bucket.NewBucketWorker")	
    defer span.End()

	sdkConfig, err :=config_aws.GetAWSConfig(ctx, bucketConfig.AwsRegion)
	if err != nil{
		return nil, err
	}

	client := s3.NewFromConfig(*sdkConfig)
	return &BucketWorker{
		client: client,
		bucketConfig: bucketConfig,
	}, nil
}

func (p *BucketWorker) GetObject(ctx context.Context, 	
								bucketNameKey 	string,
								filePath 		string,
								fileKey 		string) (*[]byte, error) {
																
	childLogger.Debug().Msg("GetObject")

	span := lib.Span(ctx, "bucket.GetObject")	
    defer span.End()

	getObjectInput := &s3.GetObjectInput{
						Bucket: aws.String(bucketNameKey+filePath),
						Key:    aws.String(fileKey),
	}

	getObjectOutput, err := p.client.GetObject(ctx, getObjectInput)
	if err != nil {
		return nil, err
	}
	defer getObjectOutput.Body.Close()

	bodyBytes, err := io.ReadAll(getObjectOutput.Body)
	if err != nil {
		return nil, err
	}

	return &bodyBytes, nil
}

func (p *BucketWorker) PutObject(ctx context.Context, 	
								fileKey string,
								file 	interface{}) error {
	childLogger.Debug().Msg("PutObject")

	span := lib.Span(ctx, "bucket.PutObject")	
    defer span.End()

	log.Debug().Interface("******** fileKey :", p.bucketConfig.BucketNameKey+p.bucketConfig.FilePath + fileKey).Msg("")

	b, err := json.Marshal(file)
    if err != nil {
        return err
    }

	file_reader := bytes.NewReader(b)

	putObjectInput := &s3.PutObjectInput{
						Bucket: aws.String(p.bucketConfig.BucketNameKey+p.bucketConfig.FilePath),
						Key:    aws.String(fileKey),
						Body: file_reader,
	}

	_, err = p.client.PutObject(ctx, putObjectInput)
	if err != nil {
		return err
	}

	return nil
}

func (p *BucketWorker) PutImageObject(ctx context.Context, 	
								fileKey string,
								file 	*[]byte) error {
	childLogger.Debug().Msg("PutObject")

	span := lib.Span(ctx, "bucket.PutObject")	
    defer span.End()

	log.Debug().Interface("******** fileKey :", p.bucketConfig.BucketNameKey+p.bucketConfig.FilePath + fileKey).Msg("")

	putObjectInput := &s3.PutObjectInput{
						Bucket: aws.String(p.bucketConfig.BucketNameKey+p.bucketConfig.FilePath),
						Key:    aws.String(fileKey),
						Body: bytes.NewReader(*file),
	}

	_, err := p.client.PutObject(ctx, putObjectInput)
	if err != nil {
		return err
	}

	return nil
}