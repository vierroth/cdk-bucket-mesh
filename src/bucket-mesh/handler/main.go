package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func handleRequest(ctx context.Context, event cfn.Event) (cfn.Response, error) {

	physicalResourceID := event.PhysicalResourceID
	if physicalResourceID == "" {
		physicalResourceID = "BucketMeshReplication"
	}

	bucketNames := event.ResourceProperties["bucketNames"].([]string)
	replicationRoleArn := event.ResourceProperties["replicationRoleArn"].(string)

	if len(bucketNames) < 2 {
		Logger.Info("Less than two buckets provided, skipping replication configuration")
		return cfn.Response{
			Status:             "SUCCESS",
			StackID:            event.StackID,
			RequestID:          event.RequestID,
			LogicalResourceID:  event.LogicalResourceID,
			PhysicalResourceID: physicalResourceID,
		}, nil
	}

	switch event.RequestType {
	case cfn.RequestCreate, cfn.RequestUpdate:
		for _, srcBucket := range bucketNames {
			var rules []types.ReplicationRule
			var priority int32 = 1

			for _, dstBucket := range bucketNames {
				if dstBucket == srcBucket {
					continue
				}

				ruleID := fmt.Sprintf("replicate-%s-to-%s", srcBucket, dstBucket)

				rules = append(rules, types.ReplicationRule{
					ID:       aws.String(ruleID),
					Status:   types.ReplicationRuleStatusEnabled,
					Priority: aws.Int32(priority),
					Filter: &types.ReplicationRuleFilter{
						Prefix: aws.String(""),
					},
					Destination: &types.Destination{
						Bucket: aws.String("arn:aws:s3:::" + dstBucket),
					},
				})

				priority++
			}

			if len(rules) == 0 {
				continue
			}

			_, err := S3.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
				Bucket: aws.String(srcBucket),
				ReplicationConfiguration: &types.ReplicationConfiguration{
					Role:  aws.String(replicationRoleArn),
					Rules: rules,
				},
			})
			if err != nil {
				Logger.Error("Failed to put bucket replication configuration",
					slog.String("bucket", srcBucket),
					slog.Any("error", err),
				)
				return cfn.Response{}, err
			}
		}

	case cfn.RequestDelete:
		for _, srcBucket := range bucketNames {
			_, err := S3.PutBucketReplication(ctx, &s3.PutBucketReplicationInput{
				Bucket: aws.String(srcBucket),
				ReplicationConfiguration: &types.ReplicationConfiguration{
					Role:  aws.String(replicationRoleArn),
					Rules: []types.ReplicationRule{},
				},
			})
			if err != nil {
				Logger.Error("Failed to clear bucket replication configuration",
					slog.String("bucket", srcBucket),
					slog.Any("error", err),
				)
				return cfn.Response{}, err
			}
		}

	default:
		return cfn.Response{}, errors.New("invalid request type")
	}

	return cfn.Response{}, nil
}

func main() {
	lambda.Start(handleRequest)
}
