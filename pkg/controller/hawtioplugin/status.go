package hawtioplugin

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	hawtiopluginv1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtioplugin/v1"
	gatewayres "github.com/hawtio/hawtio-operator/pkg/resources/hawtioplugin/gateway"
	pluginres "github.com/hawtio/hawtio-operator/pkg/resources/hawtioplugin/plugin"
	"github.com/hawtio/hawtio-operator/pkg/util"
)

func createStatusWithImages(bv util.BuildVariables) hawtiopluginv1.HawtioPluginStatus {
	return hawtiopluginv1.HawtioPluginStatus{
		PluginImage:  bv.ImageURI(bv.GetPluginImageRepo(), bv.GetPluginVersion()),
		GatewayImage: bv.ImageURI(bv.GetPluginGatewayImageRepo(), bv.GetPluginGatewayVersion()),
	}
}

// isDeploymentFailed checks if the Deployment has exceeded its progress deadline.
func isDeploymentFailed(deployment *appsv1.Deployment) bool {
	for _, cond := range deployment.Status.Conditions {
		// Check for the specific Type and Reason that indicate failure
		if cond.Type == appsv1.DeploymentProgressing && cond.Reason == "ProgressDeadlineExceeded" {
			return true
		}
	}
	return false
}

func updateStatus(deployments map[string]*appsv1.Deployment, status *hawtiopluginv1.HawtioPluginStatus) {
	readyReplicas := 0
	failed := 0
	pluginDeployment := deployments[pluginres.PluginName]
	if pluginDeployment != nil {
		status.PluginImage = pluginDeployment.Spec.Template.Spec.Containers[0].Image
		if pluginDeployment.Status.ReadyReplicas > 0 {
			readyReplicas++
		}
		if isDeploymentFailed(pluginDeployment) {
			failed++
		}
	}

	gatewayDeployment := deployments[gatewayres.GatewayName]
	if gatewayDeployment != nil {
		status.GatewayImage = gatewayDeployment.Spec.Template.Spec.Containers[0].Image
		if gatewayDeployment.Status.ReadyReplicas > 0 {
			readyReplicas++
		}
		if isDeploymentFailed(gatewayDeployment) {
			failed++
		}
	}

	// Determine the overall phase based on the deployment's readiness.
	if readyReplicas == 2 {
		status.Phase = hawtiopluginv1.PluginPhaseDeployed
	} else if failed > 0 {
		// At least one of the Deployments isn't ready yet, check if timed out.
		status.Phase = hawtiopluginv1.PluginPhaseFailed
	} else {
		// No failure but deployments not yet ready. Likely pulling images
		// or waiting for the container to start. Keep it as Initialized.
		status.Phase = hawtiopluginv1.PluginPhaseInitialized
	}
}

func (r *ReconcileHawtioPlugin) setPluginStatus(ctx context.Context, hawtioPlugin *hawtiopluginv1.HawtioPlugin, newStatus hawtiopluginv1.HawtioPluginStatus) error {
	r.Logger.V(util.DebugLogLevel).Info("Setting HawtioPlugin CR Status:", "status", newStatus)

	previous := hawtioPlugin.DeepCopy()
	hawtioPlugin.Status.Phase = newStatus.Phase
	hawtioPlugin.Status.PluginImage = newStatus.PluginImage
	hawtioPlugin.Status.GatewayImage = newStatus.GatewayImage
	err := r.Client.Status().Patch(ctx, hawtioPlugin, client.MergeFrom(previous))
	if err != nil {
		return fmt.Errorf("failed to update HawtioPlugin status: %v: %v", newStatus, err)
	}

	return nil
}
