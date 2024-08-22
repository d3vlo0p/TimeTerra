package v1alpha1

type AwsCredentialsKeysRef struct {
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	SessionKey string `json:"sessionKey,omitempty"`
}

type AwsCredentialsSpec struct {
	SecretName string                 `json:"secretName"`
	Namespace  string                 `json:"namespace,omitempty"`
	KeysRef    *AwsCredentialsKeysRef `json:"keysRef,omitempty"`
}
