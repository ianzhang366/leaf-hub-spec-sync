package bundles

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-cluster-management/leaf-hub-spec-sync/pkg/bundle"
	"github.com/open-cluster-management/leaf-hub-spec-sync/pkg/controller/helpers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LeafHubBundlesSpecSync syncs bundles spec objects.
type LeafHubBundlesSpecSync struct {
	log               logr.Logger
	k8sClient         client.Client
	bundleUpdatesChan chan *bundle.ObjectsBundle
}

// AddLeafHubBundlesSpecSync adds bundles spec syncer to the manager.
func AddLeafHubBundlesSpecSync(log logr.Logger, mgr ctrl.Manager, bundleUpdatesChan chan *bundle.ObjectsBundle) error {
	if err := mgr.Add(&LeafHubBundlesSpecSync{
		log:               log,
		k8sClient:         mgr.GetClient(),
		bundleUpdatesChan: bundleUpdatesChan,
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
			if err := helpers.UpdateObject(ctx, syncer.k8sClient, obj); err != nil {
				syncer.log.Error(err, "failed to update object", "name", obj.GetName(),
					"namespace", obj.GetNamespace(), "kind", obj.GetKind())
			} else {
				syncer.log.Info("object updated", "name", obj.GetName(), "namespace",
					obj.GetNamespace(), "kind", obj.GetKind())
			}
		}

		for _, obj := range receivedBundle.DeletedObjects {
			if deleted, err := helpers.DeleteObject(ctx, syncer.k8sClient, obj); err != nil {
				syncer.log.Error(err, "failed to delete object", "name", obj.GetName(),
					"namespace", obj.GetNamespace(), "kind", obj.GetKind())
			} else if deleted {
				syncer.log.Info("object deleted", "name", obj.GetName(), "namespace",
					obj.GetNamespace(), "kind", obj.GetKind())
			}
		}
	}
}
