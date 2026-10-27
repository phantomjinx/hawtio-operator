package hawtioplugin

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	osv1 "github.com/openshift/api/console/v1"

	hawtiov2 "github.com/hawtio/hawtio-operator/pkg/apis/hawtio/v2"
	hawtiopluginv1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtioplugin/v1"

	commonctrl "github.com/hawtio/hawtio-operator/pkg/controller/common"
	pluginres "github.com/hawtio/hawtio-operator/pkg/resources/hawtioplugin/plugin"

	"github.com/hawtio/hawtio-operator/pkg/util"
)

// HawtioPluginName is the name of the single console-plugin
const HawtioPluginName = "hawtio-console-plugin"

func (r *ReconcileHawtioPlugin) reconcileHawtio(ctx context.Context) (bool, error) {
	r.Logger.V(util.DebugLogLevel).Info("Querying Hawtio custom resources")

	hawtioList := &hawtiov2.HawtioList{}
	err := r.Client.List(ctx, hawtioList)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Logger.V(util.DebugLogLevel).Info("No Hawtio CRs found")
			return false, nil
		}
		// Error reading the object - requeue the request.
		return false, err
	}

	if len(hawtioList.Items) == 0 {
		// No hawtio CRs
		r.Logger.V(util.DebugLogLevel).Info("No Hawtio CRs found")
		return false, nil
	}

	enabled := false
	for _, h := range hawtioList.Items {
		if h.Spec.EnableHawtioPlugin != nil && *h.Spec.EnableHawtioPlugin {
			// First one that requires the console plugin
			enabled = true
			break
		}
	}

	return enabled, nil
}

func (r *ReconcileHawtioPlugin) fetchHawtioPlugin(ctx context.Context) (*hawtiopluginv1.HawtioPlugin, error) {
	r.Logger.V(util.DebugLogLevel).Info("Fetching the HawtioPlugin custom resource")

	pluginList := &hawtiopluginv1.HawtioPluginList{}
	err := r.Client.List(ctx, pluginList)
	if err != nil {
		if kerrors.IsNotFound(err) {
			r.Logger.V(util.DebugLogLevel).Info("No HawtioPlugin CR found")
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}

	if len(pluginList.Items) == 0 {
		r.Logger.V(util.DebugLogLevel).Info("No HawtioPlugin CR found")
		return nil, nil
	}

	hawtioPlugin := pluginList.Items[0]
	if len(pluginList.Items) > 1 {
		r.Logger.Info("Warning: Multiple HawtioPlugin CRs found in cluster",
			"count", len(pluginList.Items),
			"selected", hawtioPlugin.Name,
		)
	}

	return &hawtioPlugin, nil
}

func (r *ReconcileHawtioPlugin) setHawtioPluginAnnotations(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, r.Client, hawtioPlugin, func() error {
		annotations := map[string]string{
			"hawt.io/managed-by":  "hawtio-operator",
			"hawt.io/description": "Managed resource - controlled by Hawtio CRs with enableHawtioPlugin=true",
		}
		hawtioPlugin.Annotations = util.MergeMap(hawtioPlugin.Annotations, annotations)
		return nil
	})
}

func (r *ReconcileHawtioPlugin) createHawtioPlugin(ctx context.Context) error {
	hawtioPlugin := hawtiopluginv1.NewHawtioPlugin(HawtioPluginName)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hawtioPlugin, func() error {
		// Intentionally empty: spec remains empty default
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileHawtioPlugin) reconcileConsolePlugin(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin) (controllerutil.OperationResult, error) {
	namespace := r.OperatorPod.Namespace

	// The object the K8s client fetches into, and ultimately saves.
	targetConsolePlugin := pluginres.NewDefaultConsolePlugin()
	reqLogger := r.Logger.WithName(fmt.Sprintf("%s-reconcilePluginResource", targetConsolePlugin.Name))

	opResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, targetConsolePlugin, func() error {
		// A read-only copy of the cluster state for diff logging
		liveSnapshot := targetConsolePlugin.DeepCopy()

		// Set the owner reference for garbage collection
		if err := controllerutil.SetControllerReference(hawtioPlugin, targetConsolePlugin, r.Scheme); err != nil {
			return err
		}

		// Local, ideal state generated from the Hawtio CR
		blueprint, err := pluginres.NewConsolePlugin(hawtioPlugin, namespace, r.BuildVariables, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error reconciling console plugin")
			return err
		}

		serverBlueprint, err := commonctrl.HydrateDefaults(ctx, r.Client, blueprint, func(source, hydrated *osv1.ConsolePlugin) {})
		if err != nil {
			return err
		}

		targetConsolePlugin.Labels = util.MergeMap(targetConsolePlugin.Labels, blueprint.Labels)
		targetConsolePlugin.Annotations = util.MergeMap(targetConsolePlugin.Annotations, blueprint.Annotations)
		// Assign the fully hydrated and patched blueprint spec
		targetConsolePlugin.Spec = serverBlueprint.Spec

		// Report any known differences to the log (only if in debug log level)
		util.ReportDiff("Console Plugin", liveSnapshot, targetConsolePlugin)

		return nil
	})

	// There is a resource but the default client cache cannot see it
	// since it is legacy and not labelled. The create failed so it
	// should be adopted and reconciled.
	if err != nil && kerrors.IsAlreadyExists(err) {
		adoptionErr := r.AdoptLegacyResource(ctx, pluginres.NewDefaultConsolePlugin())
		if adoptionErr != nil {
			// If adoption failed (e.g., API error) or an adoption was called
			// return any of these errors
			return controllerutil.OperationResultNone, adoptionErr
		}
	}

	util.ReportResourceChange("Console Plugin", targetConsolePlugin, opResult)
	return opResult, err
}
