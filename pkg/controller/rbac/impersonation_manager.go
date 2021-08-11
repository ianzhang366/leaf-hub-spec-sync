package rbac

import (
	"encoding/base64"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

const (
	noUserIdentity         = ""
	userIdentityAnnotation = "open-cluster-management.io/user-identity"
)

// NewImpersonationManager creates a new instance of ImpersonationManager.
func NewImpersonationManager(config *rest.Config) *ImpersonationManager {
	impersonationConfigCache := make(map[string]rest.ImpersonationConfig)
	// when there is no user identity in the object, use the default controller impersonation config
	impersonationConfigCache[noUserIdentity] = config.Impersonate

	return &ImpersonationManager{
		k8sConfig:                config,
		impersonationConfigCache: make(map[string]rest.ImpersonationConfig),
	}
}

// ImpersonationManager manages the k8s clients for the various users and for the controller.
type ImpersonationManager struct {
	k8sConfig                *rest.Config
	impersonationConfigCache map[string]rest.ImpersonationConfig
}

// Impersonate gets an object and returns the k8s client that represents the request user.
// in case the user-identity header doesn't exist on the object, returns the controller k8s client.
func (manager *ImpersonationManager) Impersonate(obj *unstructured.Unstructured) error {
	userIdentity, err := manager.getUserIdentityFromObj(obj)
	if err != nil {
		return fmt.Errorf("failed to get user identity from object - %w", err)
	}

	// in case no user identity was received, it will find the default controller k8s client in the cache.
	if _, found := manager.impersonationConfigCache[userIdentity]; !found {
		manager.impersonationConfigCache[userIdentity] = rest.ImpersonationConfig{
			UserName: userIdentity,
			Groups:   nil,
			Extra:    nil,
		}
	}

	manager.k8sConfig.Impersonate = manager.impersonationConfigCache[userIdentity] // configure impersonation.

	return nil
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
