package sync

import (
	"testing"

	"encoding/json"
	"github.com/rancher/go-rancher/v3"
	"github.com/stretchr/testify/assert"
)

var (
	oneRegistryRequest = client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Image:                "nginx",
				RegistryCredentialId: "1",
			},
		},
		RegistryCredentials: []client.Credential{
			{
				Resource: client.Resource{
					Id: "1",
				},
				PublicValue: "username",
				SecretValue: "password",
			},
		},
	}
	twoRegistriesRequest = client.DeploymentSyncRequest{
		Containers: []client.Container{
			{
				Image:                "nginx",
				RegistryCredentialId: "1",
			},
			{
				Image:                "nginx",
				RegistryCredentialId: "1",
			},
			{
				Image:                "quay.io/nginx",
				RegistryCredentialId: "2",
			},
		},
		RegistryCredentials: []client.Credential{
			{
				Resource: client.Resource{
					Id: "1",
				},
				PublicValue: "username1",
				SecretValue: "password1",
			},
			{
				Resource: client.Resource{
					Id: "2",
				},
				PublicValue: "username2",
				SecretValue: "password2",
			},
		},
	}
)

func TestCredentialsFromDeploymentUnit(t *testing.T) {
	secrets := getCredentialsFromDeploymentUnit(oneRegistryRequest)
	assert.Len(t, secrets, 1)
	var data map[string]map[string]interface{}
	assert.Nil(t, json.Unmarshal(secrets[0].Data[".dockercfg"], &data))
	assert.Equal(t, data["https://index.docker.io/v1/"]["username"], "username")
	assert.Equal(t, data["https://index.docker.io/v1/"]["password"], "password")

	secrets = getCredentialsFromDeploymentUnit(twoRegistriesRequest)
	assert.Len(t, secrets, 2)
	assert.Nil(t, json.Unmarshal(secrets[0].Data[".dockercfg"], &data))
	assert.Equal(t, data["https://index.docker.io/v1/"]["username"], "username1")
	assert.Equal(t, data["https://index.docker.io/v1/"]["password"], "password1")
	assert.Nil(t, json.Unmarshal(secrets[1].Data[".dockercfg"], &data))
	assert.Equal(t, data["https://quay.io"]["username"], "username2")
	assert.Equal(t, data["https://quay.io"]["password"], "password2")
}
