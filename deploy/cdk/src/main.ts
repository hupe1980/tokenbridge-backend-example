import * as path from 'node:path';
import * as lambda_go from '@aws-cdk/aws-lambda-go-alpha';
import { App, RemovalPolicy, Stack, StackProps, CfnOutput } from 'aws-cdk-lib';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as kms from 'aws-cdk-lib/aws-kms';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import { RetentionDays } from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';

export class MyStack extends Stack {
  constructor(scope: Construct, id: string, props: StackProps = {}) {
    super(scope, id, props);

    // Create an asymmetric RSA256 KMS key
    const rsaKey = new kms.Key(this, 'RSA256-Key', {
      keySpec: kms.KeySpec.RSA_2048,
      keyUsage: kms.KeyUsage.SIGN_VERIFY,
      description: 'Asymmetric RSA256 key for signing access tokens',
      removalPolicy: RemovalPolicy.DESTROY, // Change to RETAIN for production
    });

    // Define Go Lambda functions
    const githubExchangeLambda = new lambda_go.GoFunction(this, 'GithubExchangeLambda', {
      entry: path.join(__dirname, '..', '..', '..', 'app', 'cmd', 'github'),
      environment: {
        KMS_KEY_ID: rsaKey.keyId, // Pass the KMS key ID as an environment variable
      },
      logRetention: RetentionDays.ONE_WEEK,
      architecture: lambda.Architecture.ARM_64,
    });

    const k8sExchangeLambda = new lambda_go.GoFunction(this, 'K8sExchangeLambda', {
      entry: path.join(__dirname, '..', '..', '..', 'app', 'cmd', 'k8s'),
      environment: {
        KMS_KEY_ID: rsaKey.keyId, // Pass the KMS key ID as an environment variable
      },
      logRetention: RetentionDays.ONE_WEEK,
      architecture: lambda.Architecture.ARM_64,
    });

    const jwksLambda = new lambda_go.GoFunction(this, 'JwksLambda', {
      entry: path.join(__dirname, '..', '..', '..', 'app', 'cmd', 'jwks'),
      environment: {
        KMS_KEY_ID: rsaKey.keyId, // Pass the KMS key ID as an environment variable
      },
      logRetention: RetentionDays.ONE_WEEK,
      architecture: lambda.Architecture.ARM_64,
    });

    // Grant permissions to the Lambda functions to use the KMS key
    rsaKey.grant(githubExchangeLambda, 'kms:Sign');
    rsaKey.grant(k8sExchangeLambda, 'kms:Sign');
    rsaKey.grant(jwksLambda, 'kms:GetPublicKey');

    // Create HTTP API (API Gateway v2)
    const httpApi = new apigatewayv2.HttpApi(this, 'TokenBridgeHttpApi', {
      apiName: 'TokenBridgeHttpApi',
      description: 'This is an HTTP API for token bridge',
    });

    // Add /exchange POST route
    httpApi.addRoutes({
      path: '/github/exchange',
      methods: [apigatewayv2.HttpMethod.POST],
      integration: new integrations.HttpLambdaIntegration('GithubExchangeIntegration', githubExchangeLambda),
    });

    httpApi.addRoutes({
      path: '/k8s/exchange',
      methods: [apigatewayv2.HttpMethod.POST],
      integration: new integrations.HttpLambdaIntegration('K8sExchangeIntegration', k8sExchangeLambda),
    });

    // Add /.well-known/jwks.json GET route
    httpApi.addRoutes({
      path: '/.well-known/jwks.json',
      methods: [apigatewayv2.HttpMethod.GET],
      integration: new integrations.HttpLambdaIntegration('JwksIntegration', jwksLambda),
    });

    // Configure stage-level throttling
    httpApi.addStage('default', {
      stageName: 'default',
      autoDeploy: true,
      throttle: {
        burstLimit: 100,
        rateLimit: 50,
      },
    });

    // Output the API Gateway URL
    new CfnOutput(this, 'HttpApiUrl', {
      value: httpApi.apiEndpoint,
      description: 'The URL of the HTTP API Gateway',
    });
  }
}

// Development environment
const devEnv = {
  account: '741759823656', //process.env.CDK_DEFAULT_ACCOUNT,
  region: 'eu-central-1', //process.env.CDK_DEFAULT_REGION,
};

const app = new App();

new MyStack(app, 'aws-tokenbridge-backend-dev', { env: devEnv });

app.synth();
