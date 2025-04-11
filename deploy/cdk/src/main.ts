import * as path from 'node:path';
import { App, RemovalPolicy, Stack, StackProps, CfnOutput } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as lambda_go from '@aws-cdk/aws-lambda-go-alpha';
import * as kms from 'aws-cdk-lib/aws-kms';
import { RetentionDays } from 'aws-cdk-lib/aws-logs';

export class MyStack extends Stack {
  constructor(scope: Construct, id: string, props: StackProps = {}) {
    super(scope, id, props);

    // Create an asymmetric RSA256 KMS key
    const rsaKey = new kms.Key(this, 'RSA256Key', {
      keySpec: kms.KeySpec.RSA_2048,
      keyUsage: kms.KeyUsage.SIGN_VERIFY,
      description: 'Asymmetric RSA256 key for signing access tokens',
      removalPolicy: RemovalPolicy.DESTROY, // Change to RETAIN for production
    });

    // Define Go Lambda functions
    const exchangeLambda = new lambda_go.GoFunction(this, 'ExchangeLambda', {
      entry: path.join(__dirname, '..', '..', '..', 'app', 'cmd', 'exchange'),
      environment: {
        KMS_KEY_ID: rsaKey.keyId, // Pass the KMS key ID as an environment variable
      },
      logRetention: RetentionDays.ONE_WEEK,
    });

    const jwksLambda = new lambda_go.GoFunction(this, 'JwksLambda', {
      entry: path.join(__dirname, '..', '..', '..', 'app', 'cmd', 'jwks'),
      environment: {
        KMS_KEY_ID: rsaKey.keyId, // Pass the KMS key ID as an environment variable
      },
      logRetention: RetentionDays.ONE_WEEK,
    });

    // Grant permissions to the Lambda functions to use the KMS key
    rsaKey.grant(exchangeLambda, 'kms:Sign');
    rsaKey.grant(jwksLambda, 'kms:GetPublicKey');

    // Create HTTP API (API Gateway v2)
    const httpApi = new apigatewayv2.HttpApi(this, 'TokenBridgeHttpApi', {
      apiName: 'TokenBridgeHttpApi',
      description: 'This is an HTTP API for token bridge',
    });

    // Add /exchange POST route
    httpApi.addRoutes({
      path: '/exchange',
      methods: [apigatewayv2.HttpMethod.POST],
      integration: new integrations.HttpLambdaIntegration('ExchangeIntegration', exchangeLambda),
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
