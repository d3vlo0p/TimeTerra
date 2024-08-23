package controller

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/credentials"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func getAwsCredentialProviderFromSecret(secret *corev1.Secret, keysRef *v1alpha1.AwsCredentialsKeysRef) (*credentials.StaticCredentialsProvider, error) {
	values, err := decodeSecret(secret)
	if err != nil {
		return nil, err
	}

	var kAccessKeyId, kSecretAccessKey, kSessionToken = "aws_access_key_id", "aws_secret_access_key", "aws_session_token"
	// override default keys if specified
	if keysRef != nil {
		kAccessKeyId, kSecretAccessKey, kSessionToken = keysRef.AccessKey, keysRef.SecretKey, keysRef.SessionKey
	}

	accessKeyId, ok := values[kAccessKeyId]
	if !ok {
		return nil, fmt.Errorf("failed to get AccessKeyId from Secret resource, no %q key found", kAccessKeyId)
	}
	secretAccessKey, ok := values[kSecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("failed to get SecretAccessKey from Secret resource, no %q key found", kSecretAccessKey)
	}
	sessionToken := ""
	if kSessionToken != "" {
		sessionToken, ok = values[kSessionToken]
		if !ok && keysRef != nil {
			return nil, fmt.Errorf("failed to get SessionToken from Secret resource, no %q key found", kSessionToken)
		}
	}

	r := credentials.NewStaticCredentialsProvider(
		accessKeyId,
		secretAccessKey,
		sessionToken,
	)

	return &r, nil
}
