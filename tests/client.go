package integration

import "github.com/rancher/go-rancher/v3"

type mockClient struct{}

func (c *mockClient) GetOpts() client.ClientOpts {
	return client.ClientOpts{}
}

func (c *mockClient) Publish(pubish *client.Publish) error {
	return nil
}

func (c *mockClient) GetCluster(id string) (*client.Cluster, error) {
	return &client.Cluster{
		Resource: client.Resource{
			Id: "1c1",
		},
		K8sClientConfig: &client.K8sClientConfig{
			Address: kubernetesAddress,
		},
	}, nil
}
