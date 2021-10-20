package helpers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName      = "leaf-hub-spec-sync"
	notFoundErrorSuffix = "not found"

	targetNamespace = "hub-of-hubs.open-cluster-management.io/remoteNamespace"
)

// UpdateObject function updates a given k8s object.
func UpdateObject(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	objectBytes, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to update object - %w", err)
	}

	if err := ensureTargetNamespace; err != nil {
		return fmt.Errorf("failed to create target namespace, err: %w", err)
	}

	forceChanges := true

	if err := k8sClient.Patch(ctx, obj, client.RawPatch(types.ApplyPatchType, objectBytes), &client.PatchOptions{
		FieldManager: controllerName,
		Force:        &forceChanges,
	}); err != nil {
		return fmt.Errorf("failed to update object - %w", err)
	}

	return nil
}

// DeleteObject tries to delete the given object from k8s. returns error and true/false if object was deleted or not.
func DeleteObject(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) (bool, error) {
	if err := k8sClient.Delete(ctx, obj); err != nil {
		if !strings.HasSuffix(err.Error(), notFoundErrorSuffix) {
			return false, fmt.Errorf("failed to delete object - %w", err)
		}

		return false, nil
	}

	return true, nil
}

// ensureTargetNamespace creates a target namespace if the object's namespace doesn't exist on leaf hub, mainly for clusterlifecycle
func ensureTargetNamespace(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	tNs := obj.GetName()

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tNs}}

	if err := k8sClient.Get(ctx, ns); err != nil {
		return nil
	}

	return k8sClient.Create(ctx, ns)
}
