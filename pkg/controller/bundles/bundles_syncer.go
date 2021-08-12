package bundles

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-cluster-management/leaf-hub-spec-sync/pkg/bundle"
	"github.com/open-cluster-management/leaf-hub-spec-sync/pkg/controller/helpers"
	"github.com/open-cluster-management/leaf-hub-spec-sync/pkg/controller/rbac"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	updateOperation      = "update"
	deleteOperation      = "delete"
	impersonateOperation = "impersonate"
)

// LeafHubBundlesSpecSync syncs bundles spec objects.
type LeafHubBundlesSpecSync struct {
	log                  logr.Logger
	impersonationManager *rbac.ImpersonationManager
	bundleUpdatesChan    chan *bundle.ObjectsBundle
}

// AddLeafHubBundlesSpecSync adds bundles spec syncer to the manager.
func AddLeafHubBundlesSpecSync(log logr.Logger, mgr ctrl.Manager, bundleUpdatesChan chan *bundle.ObjectsBundle) error {
	impersonationManager, err := rbac.NewImpersonationManager()
	if err != nil {
		return fmt.Errorf("failed to initialize impersonation manager - %w", err)
	}

	if err := mgr.Add(&LeafHubBundlesSpecSync{
		log:                  log,
		impersonationManager: impersonationManager,
		bundleUpdatesChan:    bundleUpdatesChan,
	}); err != nil {
		return fmt.Errorf("failed to add bundles spec syncer - %w", err)
	}

	return nil
}

// Start function starts bundles spec syncer.
func (syncer *LeafHubBundlesSpecSync) Start(stopChannel <-chan struct{}) error {
	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	go syncer.sync(ctx)

	for {
		<-stopChannel // blocking wait for stop event
		syncer.log.Info("stopped bundles syncer")
		cancelContext()

		return nil
	}
}

func (syncer *LeafHubBundlesSpecSync) sync(ctx context.Context) {
	syncer.log.Info("start bundles syncing...")

	for {
		receivedBundle := <-syncer.bundleUpdatesChan
		for _, obj := range receivedBundle.Objects {
			k8sClient, err := syncer.impersonationManager.Impersonate(obj)
			if err != nil {
				syncer.logFailure(err, obj, impersonateOperation)
				continue
			}

			if err := helpers.UpdateObject(ctx, k8sClient, obj); err != nil {
				syncer.logFailure(err, obj, updateOperation)
			} else {
				syncer.log.Info("object updated", "name", obj.GetName(), "namespace",
					obj.GetNamespace(), "kind", obj.GetKind())
			}
		}

		for _, obj := range receivedBundle.DeletedObjects {
			k8sClient, err := syncer.impersonationManager.Impersonate(obj)
			if err != nil {
				syncer.logFailure(err, obj, impersonateOperation)
				continue
			}

			if deleted, err := helpers.DeleteObject(ctx, k8sClient, obj); err != nil {
				syncer.logFailure(err, obj, deleteOperation)
			} else if deleted {
				syncer.log.Info("object deleted", "name", obj.GetName(), "namespace",
					obj.GetNamespace(), "kind", obj.GetKind())
			}
		}
	}
}

func (syncer *LeafHubBundlesSpecSync) logFailure(err error, obj *unstructured.Unstructured, operation string) {
	syncer.log.Error(err, fmt.Sprintf("failed to %s", operation), "name", obj.GetName(),
		"namespace", obj.GetNamespace(), "kind", obj.GetKind())
}
