package controller

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/credentials"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func getAwsCredentialProviderFromSecret(secret *corev1.Secret, keysRef *v1alpha1.AwsCredentialsKeysRef) (*credentials.StaticCredentialsProvider, error) {
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret resource contains no data")
	}

	var kAccessKeyId, kSecretAccessKey, kSessionToken = "aws_access_key_id", "aws_secret_access_key", "aws_session_token"
	// override default keys if specified
	if keysRef != nil {
		if keysRef.AccessKey != "" {
			kAccessKeyId = keysRef.AccessKey
		}
		if keysRef.SecretKey != "" {
			kSecretAccessKey = keysRef.SecretKey
		}
		if keysRef.SessionKey != "" {
			kSessionToken = keysRef.SessionKey
		} else {
			kSessionToken = ""
		}
	}

	accessKeyIdBytes, ok := secret.Data[kAccessKeyId]
	if !ok {
		return nil, fmt.Errorf("failed to get AccessKeyId from Secret resource, no %q key found", kAccessKeyId)
	}
	secretAccessKeyBytes, ok := secret.Data[kSecretAccessKey]
	if !ok {
		return nil, fmt.Errorf("failed to get SecretAccessKey from Secret resource, no %q key found", kSecretAccessKey)
	}
	sessionToken := ""
	if kSessionToken != "" {
		sessionTokenBytes, ok := secret.Data[kSessionToken]
		if !ok && keysRef != nil {
			return nil, fmt.Errorf("failed to get SessionToken from Secret resource, no %q key found", kSessionToken)
		}
		if ok {
			sessionToken = string(sessionTokenBytes)
		}
	}

	r := credentials.NewStaticCredentialsProvider(
		string(accessKeyIdBytes),
		string(secretAccessKeyBytes),
		sessionToken,
	)

	return &r, nil
}
