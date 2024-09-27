package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func resourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}

func conditionTypeForAction(name string) string {
	return fmt.Sprintf("Action-%s", name)
}

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func decodeSecret(secret *corev1.Secret) (map[string]string, error) {
	values := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		// log.Log.Info("decoding secret", "key", k, "value", v, "dec", string(v))

		// seems that something does already base64 decoding
		// because if i print "v" in the logs, i see the base64 string
		// but if i log string(v), i see the decoded string
		values[k] = string(v)

		// dst := make([]byte, base64.StdEncoding.DecodedLen(len(v)))
		// n, err := base64.StdEncoding.Decode(dst, v)
		// if err != nil {
		// 	return nil, err
		// }
		// Base64 decoding the bytes into a slice works correctly,
		// but attempting to cast the slice to a string fails.
		// This is likely because the string conversion is trying
		// to decode the byte slice again, which is not necessary
		// since it has already been decoded into the slice.
		// values[k] = string(dst[:n])
	}
	// log.Log.Info("decoded secret", "values", values)
	return values, nil
}

func defaultNamespace(operatorNamespace string, namespace string) string {
	if namespace == "" {
		return operatorNamespace
	}
	return namespace
}
