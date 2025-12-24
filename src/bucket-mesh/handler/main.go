package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Bucket struct {
	name   string
	region string
}

func handleRequest(ctx context.Context, event cfn.Event) (cfn.Response, error) {

	physicalResourceID := event.PhysicalResourceID
	if physicalResourceID == "" {
		physicalResourceID = "BucketMeshReplication"
	}

	replicationRoleArn := event.ResourceProperties["replicationRoleArn"].(string)

	rawBuckets, ok := event.ResourceProperties["buckets"].([]any)
	if !ok {
		return cfn.Response{}, fmt.Errorf("buckets is missing or not an array")
	}

	buckets := make([]Bucket, len(rawBuckets))
	for i, rb := range rawBuckets {
		m, ok := rb.(map[string]any)
		if !ok {
			return cfn.Response{}, fmt.Errorf("buckets[%d] is not an object", i)
		}

		name, ok := m["name"].(string)
		if !ok {
			return cfn.Response{}, fmt.Errorf("buckets[%d].name is missing or not a string", i)
		}

		region, ok := m["region"].(string)
		if !ok {
			return cfn.Response{}, fmt.Errorf("buckets[%d].region is missing or not a string", i)
		}

		buckets[i] = Bucket{
			name:   name,
			region: region,
		}
	}

	if len(buckets) < 2 {
		Logger.Info("Less than two buckets provided, skipping replication configuration")
		return cfn.Response{
			Status:             "SUCCESS",
			StackID:            event.StackID,
			RequestID:          event.RequestID,
			LogicalResourceID:  event.LogicalResourceID,
			PhysicalResourceID: physicalResourceID,
			Data:               map[string]any{},
		}, nil
	}

	switch event.RequestType {
	case cfn.RequestCreate, cfn.RequestUpdate:
		for _, srcBucket := range buckets {
			var rules []types.ReplicationRule

			for i, dstBucket := range buckets {
				if dstBucket.name == srcBucket.name {
					continue
				}

				ruleID := fmt.Sprintf("replicate-%s-to-%s", srcBucket.name, dstBucket.name)

				rules = append(rules, types.ReplicationRule{
					ID:       aws.String(ruleID),
					Status:   types.ReplicationRuleStatusEnabled,
					Priority: aws.Int32(int32(i) + 1),
					Filter: &types.ReplicationRuleFilter{
						Prefix: aws.String(""),
					},
					Destination: &types.Destination{
						Bucket: aws.String("arn:aws:s3:::" + dstBucket.name),
						AccessControlTranslation: &types.AccessControlTranslation{
							Owner: types.OwnerOverrideDestination,
						},
					},
					DeleteMarkerReplication: &types.DeleteMarkerReplication{
						Status: types.DeleteMarkerReplicationStatusEnabled,
					},
				})
			}

			if len(rules) == 0 {
				continue
			}

			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(srcBucket.region))
			if err != nil {
				return cfn.Response{}, err
			}

			S3Client := s3.NewFromConfig(cfg)

			_, err = S3Client.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
				Bucket: aws.String(srcBucket.name),
				ReplicationConfiguration: &types.ReplicationConfiguration{
					Role:  aws.String(replicationRoleArn),
					Rules: rules,
				},
			})
			if err != nil {
				Logger.Error("Failed to put bucket replication configuration",
					slog.String("bucket", srcBucket.name),
					slog.Any("error", err),
				)
				return cfn.Response{}, err
			}
		}

		return cfn.Response{
			Status:             "SUCCESS",
			StackID:            event.StackID,
			RequestID:          event.RequestID,
			LogicalResourceID:  event.LogicalResourceID,
			PhysicalResourceID: physicalResourceID,
			Data:               map[string]any{},
		}, nil

	case cfn.RequestDelete:
		for _, srcBucket := range buckets {
			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(srcBucket.region))
			if err != nil {
				return cfn.Response{}, err
			}

			S3Client := s3.NewFromConfig(cfg)

			_, err = S3Client.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
				Bucket: aws.String(srcBucket.name),
				ReplicationConfiguration: &types.ReplicationConfiguration{
					Role:  aws.String(replicationRoleArn),
					Rules: []types.ReplicationRule{},
				},
			})
			if err != nil {
				Logger.Error("Failed to clear bucket replication configuration",
					slog.String("bucket", srcBucket.name),
					slog.Any("error", err),
				)
				return cfn.Response{}, err
			}
		}

		return cfn.Response{
			Status:             "SUCCESS",
			StackID:            event.StackID,
			RequestID:          event.RequestID,
			LogicalResourceID:  event.LogicalResourceID,
			PhysicalResourceID: physicalResourceID,
			Data:               map[string]any{},
		}, nil

	default:
		return cfn.Response{}, errors.New("invalid request type")
	}
}

func main() {
	lambda.Start(handleRequest)
}
