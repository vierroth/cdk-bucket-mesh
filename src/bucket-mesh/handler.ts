import {
	S3Client,
	PutBucketReplicationCommand,
	DeleteBucketReplicationCommand,
	ReplicationRule,
} from "@aws-sdk/client-s3";

interface Bucket {
	name: string;
	region: string;
	accountId: string;
}

interface OnEventRequest {
	RequestType: "Create" | "Update" | "Delete";
	PhysicalResourceId?: string;
	ResourceProperties: {
		buckets: Bucket[];
		replicationRoleArn: string;
	};
	LogicalResourceId: string;
	StackId: string;
	RequestId: string;
	ResourceType: string;
}

interface OnEventResponse {
	PhysicalResourceId: string;
	Data: Record<string, unknown>;
}

export const handler = async (
	event: OnEventRequest,
): Promise<OnEventResponse> => {
	const physicalResourceId =
		event.PhysicalResourceId ?? "BucketMeshReplication";
	const { buckets, replicationRoleArn } = event.ResourceProperties;

	if (buckets.length < 2) {
		console.log(
			"Less than two buckets provided, skipping replication configuration",
		);
		return { PhysicalResourceId: physicalResourceId, Data: {} };
	}

	switch (event.RequestType) {
		case "Create":
		case "Update": {
			for (const srcBucket of buckets) {
				const rules: ReplicationRule[] = [];

				buckets.forEach((dstBucket, i) => {
					if (dstBucket.name === srcBucket.name) return;
					rules.push({
						ID: `replicate-${srcBucket.name}-to-${dstBucket.name}`,
						Status: "Enabled",
						Priority: i + 1,
						Filter: { Prefix: "" },
						Destination: {
							Bucket: `arn:aws:s3:::${dstBucket.name}`,
							Account: dstBucket.accountId,
							AccessControlTranslation: { Owner: "Destination" },
						},
						DeleteMarkerReplication: { Status: "Enabled" },
					});
				});

				if (rules.length === 0) continue;

				const client = new S3Client({ region: srcBucket.region });
				await client.send(
					new PutBucketReplicationCommand({
						Bucket: srcBucket.name,
						ReplicationConfiguration: {
							Role: replicationRoleArn,
							Rules: rules,
						},
					}),
				);
			}
			return { PhysicalResourceId: physicalResourceId, Data: {} };
		}

		case "Delete": {
			for (const srcBucket of buckets) {
				const client = new S3Client({ region: srcBucket.region });
				await client.send(
					new DeleteBucketReplicationCommand({ Bucket: srcBucket.name }),
				);
			}
			return { PhysicalResourceId: physicalResourceId, Data: {} };
		}

		default:
			throw new Error(`invalid request type: ${event.RequestType}`);
	}
};
