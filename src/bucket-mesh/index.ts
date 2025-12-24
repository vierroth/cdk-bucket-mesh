import { Construct } from "constructs";
import { CustomResource, Stack } from "aws-cdk-lib";
import { Bucket } from "aws-cdk-lib/aws-s3";
import {
	Effect,
	PolicyStatement,
	Role,
	ServicePrincipal,
} from "aws-cdk-lib/aws-iam";

import { Provider } from "./provider";

interface BucketMeshResourceProps {
	readonly buckets: Bucket[];
	readonly role: Role;
}

class BucketMeshResource extends CustomResource {
	constructor(scope: Construct, id: string, props: BucketMeshResourceProps) {
		super(scope, id, {
			resourceType: "Custom::BucketMesh",
			serviceToken: Provider.getOrCreate(scope, props.buckets, props.role),
			properties: {
				buckets: props.buckets.map((b) => ({
					name: b.bucketName,
					region: Stack.of(b).region,
				})),
				replicationRoleArn: props.role.roleArn,
			},
		});
	}
}

export interface BucketMeshProps {
	readonly buckets: Bucket[];
}

/**
 * @category Constructs
 */
export class BucketMesh extends Construct {
	constructor(scope: Construct, id: string, props: BucketMeshProps) {
		super(scope, id);

		const replicationRole = new Role(this, "BucketReplicationRole", {
			assumedBy: new ServicePrincipal("s3.amazonaws.com"),
		});

		replicationRole.addToPolicy(
			new PolicyStatement({
				effect: Effect.ALLOW,
				actions: ["s3:GetReplicationConfiguration", "s3:ListBucket"],
				resources: props.buckets.map(({ bucketArn }) => bucketArn),
			}),
		);

		replicationRole.addToPolicy(
			new PolicyStatement({
				effect: Effect.ALLOW,
				actions: [
					"s3:GetObjectVersion",
					"s3:GetObjectVersionAcl",
					"s3:GetObjectVersionTagging",
					"s3:GetObjectVersionForReplication",
					"s3:GetObjectRetention",
					"s3:GetObjectLegalHold",
					"s3:ReplicateObject",
					"s3:ReplicateDelete",
					"s3:ReplicateTags",
					"s3:ObjectOwnerOverrideToBucketOwner",
				],
				resources: props.buckets.map(({ bucketArn }) => `${bucketArn}/*`),
			}),
		);

		new BucketMeshResource(this, "Resource", {
			buckets: props.buckets,
			role: replicationRole,
		});
	}
}
