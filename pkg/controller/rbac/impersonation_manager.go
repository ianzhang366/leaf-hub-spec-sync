package rbac

import (
	"encoding/base64"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	noUserIdentity         = ""
	userIdentityAnnotation = "open-cluster-management.io/user-identity"
)

// NewImpersonationManager creates a new instance of ImpersonationManager.
func NewImpersonationManager() (*ImpersonationManager, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in cluster kubeconfig - %w", err)
	}

	controllerK8sClient, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize k8s client - %w", err)
	}

	k8sClientsCache := make(map[string]client.Client)
	// when there is no user identity in the object, use the default controller impersonation config
	k8sClientsCache[noUserIdentity] = controllerK8sClient

	return &ImpersonationManager{
		k8sConfig:       config,
		k8sClientsCache: k8sClientsCache,
	}, nil
}

// ImpersonationManager manages the k8s clients for the various users and for the controller.
type ImpersonationManager struct {
	k8sConfig       *rest.Config
	k8sClientsCache map[string]client.Client
}

// Impersonate gets an object and returns the k8s client that represents the request user.
// in case the user-identity header doesn't exist on the object, returns the controller k8s client.
func (manager *ImpersonationManager) Impersonate(obj *unstructured.Unstructured) (client.Client, error) {
	userIdentity, err := manager.getUserIdentityFromObj(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to get user identity from object - %w", err)
	}

	// in case no user identity was received, it will find the default controller k8s client in the cache.
	if _, found := manager.k8sClientsCache[userIdentity]; !found {
		newConfig := rest.CopyConfig(manager.k8sConfig)
		newConfig.Impersonate = rest.ImpersonationConfig{
			UserName: userIdentity,
			Groups:   nil,
			Extra:    nil,
		}

		userK8sClient, err := client.New(newConfig, client.Options{})
		if err != nil {
			return nil, fmt.Errorf("failed to create new k8s client for user - %w", err)
		}

		manager.k8sClientsCache[userIdentity] = userK8sClient
	}

	return manager.k8sClientsCache[userIdentity], nil
}

// returns an empty string in case the user-identity annotation doesn't exist in the object.
func (manager *ImpersonationManager) getUserIdentityFromObj(obj *unstructured.Unstructured) (string, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil { // there are no annotations defined, therefore user identity annotation is not defined.
		return noUserIdentity, nil
	}

	if base64UserIdentity, found := annotations[userIdentityAnnotation]; found { // if annotation exists
		decodedUserIdentity, err := base64.StdEncoding.DecodeString(base64UserIdentity)
		if err != nil {
			return noUserIdentity, fmt.Errorf("failed to decode base64 user identity - %w", err)
		}

		return string(decodedUserIdentity), nil
	}
	// otherwise, the annotation doesn't exist
	return noUserIdentity, nil
}
