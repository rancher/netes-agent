package sync

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"

	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes-agent/utils"
)

const (
	dockerConfigKey = ".dockercfg"
	dockerRegistry  = "https://index.docker.io/v1/"
)

func credentialsFromDeploymentUnit(deploymentUnit client.DeploymentSyncRequest) []v1.Secret {
	var imagePullCredentials []v1.Secret
	for _, registryCredential := range deploymentUnit.RegistryCredentials {
		// TODO: remove hard-coded registry URL
		url := dockerRegistry
		username := registryCredential.PublicValue
		password := registryCredential.SecretValue
		imagePullCredentials = append(imagePullCredentials, v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      getSecretName(url, username, password),
				Namespace: deploymentUnit.Namespace,
			},
			Type: v1.SecretTypeDockercfg,
			Data: map[string][]byte{
				dockerConfigKey: getSecretValue(url, username, password),
			},
		})
	}
	return imagePullCredentials
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
