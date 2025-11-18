import { Construct } from "constructs";
import { CustomResource } from "aws-cdk-lib";
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
			serviceToken: Provider.getOrCreate(scope, props.buckets),
			properties: {
				bucketNames: props.buckets.map(({ bucketName }) => bucketName),
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

		new BucketMeshResource(this, "Resource", {
			buckets: props.buckets,
			role: replicationRole,
		});
	}
}
