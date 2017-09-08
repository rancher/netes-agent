package sync

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

func reconcileSecrets(clientset *kubernetes.Clientset, namespace string, secrets []v1.Secret) error {
	for _, secret := range secrets {
		if _, err := clientset.Secrets(namespace).Create(&secret); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}
