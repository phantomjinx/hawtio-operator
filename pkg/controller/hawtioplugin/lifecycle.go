package hawtioplugin

import (
	"context"
	"fmt"

	osv1 "github.com/openshift/api/console/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	hawtiopluginv1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtioplugin/v1"
	"github.com/hawtio/hawtio-operator/pkg/util"
)

const hawtioPluginFinalizer = "plugin.hawt.io/finalizer"

//
// Handles clean up if user explicity removed the HawtioPlugin CR
//
func (r *ReconcileHawtioPlugin) handleDeletion(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin) (bool, error) {
	if hawtioPlugin == nil || hawtioPlugin.GetDeletionTimestamp() == nil {
		return false, nil
	}

	r.Logger.Info("HawtioPlugin deletion requested - performing cleanup")

	// // Cleanup resources without owner references
	// err := r.cleanupPluginResources(ctx, hawtioPlugin)
	// if err != nil {
	// 	return false, err
	// }

	// Remove finalizer to allow deletion
	if controllerutil.ContainsFinalizer(hawtioPlugin, hawtioPluginFinalizer) {
		previous := hawtioPlugin.DeepCopy()
		controllerutil.RemoveFinalizer(hawtioPlugin, hawtioPluginFinalizer)
		err := r.Client.Patch(ctx, hawtioPlugin, client.MergeFrom(previous))
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *ReconcileHawtioPlugin) addFinalizer(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin) (bool, error) {
	// Add a finalizer, that's needed to clean up cluster-wide resources
	if controllerutil.ContainsFinalizer(hawtioPlugin, hawtioPluginFinalizer) {
		r.Logger.V(util.DebugLogLevel).Info("Finalizer already present")
		return false, nil // nothing to do
	}

	r.Logger.V(util.DebugLogLevel).Info("Adding a finalizer")

	previous := hawtioPlugin.DeepCopy()
	controllerutil.AddFinalizer(hawtioPlugin, hawtioPluginFinalizer)

	// Use patch rather than update to ensure only the
	// explicit changes are merged in rather than potentially
	// overwriting with a stale hawtio CR
	err := r.Client.Patch(ctx, hawtioPlugin, client.MergeFrom(previous))
	if err != nil {
		r.Logger.V(util.DebugLogLevel).Info("Error finalizer")
		return false, fmt.Errorf("failed to update finalizer: %v", err)
	}

	r.Logger.V(util.DebugLogLevel).Info("Completed finalizer")
	return true, nil
}

func (r *ReconcileHawtioPlugin) removeInstall(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin) error {
	// Target kinds provisioned by HawtioPlugin
	targetLists := []client.ObjectList{
		&corev1.ServiceList{},
		&corev1.SecretList{},
		&appsv1.DeploymentList{},
		&osv1.ConsolePluginList{},
	}

	// Query matching field indexer registered on "metadata.ownerReferences.uid"
	listOpts := []client.ListOption{
		client.InNamespace(hawtioPlugin.Namespace),
		client.MatchingFields{"metadata.ownerReferences.uid": string(hawtioPlugin.UID)},
	}

	for _, list := range targetLists {
		if err := r.Client.List(ctx, list, listOpts...); err != nil {
			return err
		}

		// Unpacks resources generically from the list wrapper
		resources, err := meta.ExtractList(list)
		if err != nil {
			return err
		}

		for _, resource := range resources {
			obj, ok := resource.(client.Object)
			if !ok {
				continue
			}

			if err := r.Client.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}
