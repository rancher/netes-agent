package sync

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/reference"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/utils"
)

const (
	dockerConfigKey = ".dockercfg"
)

func getCredentialsFromDeploymentUnit(deploymentUnit client.DeploymentSyncRequest) []v1.Secret {
	registryCredentialsMap := map[string]client.Credential{}
	for _, registryCredential := range deploymentUnit.RegistryCredentials {
		registryCredentialsMap[registryCredential.Id] = registryCredential
	}

	var imagePullCredentials []v1.Secret
	secretNames := map[string]bool{}
	for _, container := range deploymentUnit.Containers {
		registryUrl := getRegistryUrlFromImage(container.Image)
		if registryUrl == "" {
			continue
		}
		registryCredential, ok := registryCredentialsMap[container.RegistryCredentialId]
		if !ok {
			continue
		}
		secret := getSecret(registryCredential, registryUrl, deploymentUnit.Namespace)
		if _, ok := secretNames[secret.Name]; !ok {
			imagePullCredentials = append(imagePullCredentials, secret)
			secretNames[secret.Name] = true
		}
	}
	return imagePullCredentials
}

func getRegistryUrlFromImage(image string) string {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return ""
	}
	domain := reference.Domain(named)
	if domain == "docker.io" {
		return "https://index.docker.io/v1/"
	}
	return fmt.Sprintf("https://%s", domain)
}

func getSecret(registryCredential client.Credential, url, namespace string) v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSecretName(url, registryCredential.PublicValue, registryCredential.SecretValue),
			Namespace: namespace,
		},
		Type: v1.SecretTypeDockercfg,
		Data: map[string][]byte{
			dockerConfigKey: getSecretValue(url, registryCredential.PublicValue, registryCredential.SecretValue),
		},
	}
}

func getSecretValue(url, username, password string) []byte {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	value, _ := json.Marshal(map[string]interface{}{
		url: map[string]string{
			"username": username,
			"password": password,
			"auth":     auth,
		},
	})
	return value
}

func getSecretName(url, username, password string) string {
	return utils.Hash(url + username + password)
}
