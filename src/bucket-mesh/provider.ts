import { Construct } from "constructs";
import { Stack } from "aws-cdk-lib";
import { Provider as AwsProvider } from "aws-cdk-lib/custom-resources";
import { join } from "path";
import { Bucket } from "aws-cdk-lib/aws-s3";
import { Effect, PolicyStatement, Role } from "aws-cdk-lib/aws-iam";

import { LambdaBase } from "../lambda-base";

export class Provider extends AwsProvider {
	constructor(scope: Construct, id: string) {
		super(scope, id, {
			onEventHandler: new LambdaBase(scope, `${id}Handler`, {
				entry: join(__dirname, "./handler").replace("/dist/", "/src/"),
			}),
		});
	}
	static getOrCreate(
		scope: Construct,
		buckets: Bucket[],
		replicationRole: Role,
	) {
		const stack = Stack.of(scope);
		const id = "_BucketMeshProvider";
		const provider =
			(stack.node.tryFindChild(id) as Provider) || new Provider(stack, id);

		provider.onEventHandler.addToRolePolicy(
			new PolicyStatement({
				effect: Effect.ALLOW,
				actions: [
					"s3:PutReplicationConfiguration",
					"s3:DeleteBucketReplication",
				],
				resources: buckets.map(({ bucketArn }) => bucketArn),
			}),
		);

		provider.onEventHandler.addToRolePolicy(
			new PolicyStatement({
				effect: Effect.ALLOW,
				actions: ["iam:PassRole"],
				resources: [replicationRole.roleArn],
				conditions: {
					StringEquals: { "iam:PassedToService": "s3.amazonaws.com" },
				},
			}),
		);

		return provider.serviceToken;
	}
}
