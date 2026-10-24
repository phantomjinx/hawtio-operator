package plugin

import (
	"fmt"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelUtils "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	hawtiopluginv1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtioplugin/v1"
	commonres "github.com/hawtio/hawtio-operator/pkg/resources/common"
	pluginres "github.com/hawtio/hawtio-operator/pkg/resources/hawtioplugin"

	"github.com/hawtio/hawtio-operator/pkg/util"
)

const (
	deploymentSSLPort = 9443

	// Serving Certificate
	serviceSigningSecretVolumeName = "hawtio-online-console-plugin-serving"
)

func NewDefaultDeployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PluginName,
			Namespace: namespace,
		},
	}
}

func NewDeployment(hawtioPlugin *hawtiopluginv1.HawtioPlugin, namespace string, buildVariables util.BuildVariables, log logr.Logger) (*appsv1.Deployment, error) {
	log.V(util.DebugLogLevel).Info("Reconciling deployment")

	podTemplateSpec, err := newPodTemplateSpec(hawtioPlugin, buildVariables, log)
	if err != nil {
		return nil, err
	}
	return newDeployment(hawtioPlugin, namespace, buildVariables, podTemplateSpec, log)
}

func deploymentLabels(buildVariables util.BuildVariables) (map[string]string, error) {
	additionalLabels, err := labelUtils.ConvertSelectorToLabelsMap(buildVariables.AdditionalLabels)
	if err != nil {
		return nil, err
	}

	/*
	 * labels:
	 *   app: hawtio
	 * 	 deployment: hawtio-online-console-plugin
	 *   app.kubernetes.io/component: hawtio-online-console-plugin
	 *   app.kubernetes.io/instance: hawtio-online-console-plugin
	 *   app.kubernetes.io/part-of: hawtio
	 */
	labels := commonres.LabelsForHawtio(pluginName)
	labels["app.kubernetes.io/component"] = pluginName
	labels["app.kubernetes.io/instance"] = pluginName
	labels["app.kubernetes.io/part-of"] = commonres.LabelAppValue

	for name, value := range additionalLabels {
		labels[name] = value
	}

	return labels, nil
}

func newDeployment(hawtioPlugin *hawtiopluginv1.HawtioPlugin, namespace string, buildVariables util.BuildVariables, pts corev1.PodTemplateSpec, log logr.Logger) (*appsv1.Deployment, error) {
	log.V(util.DebugLogLevel).Info("New deployment")

	/*
	 * apiVersion: apps/v1
	 * kind: Deployment
	 * metadata:
	 *   labels:
	 *     app: hawtio
	 *     deployment: hawtio-online-console-plugin
	 *     app.kubernetes.io/component: hawtio-online-console-plugin
	 *     app.kubernetes.io/instance: hawtio-online-console-plugin
	 *     app.kubernetes.io/part-of: hawtio
	 *   name: hawtio-online-console-plugin
	 *   namespace: <operator namespace>
	 * spec:
	 *   selector:
	 *     matchLabels:
	 *       app: hawtio
	 *       deployment: hawtio-online-console-plugin
	 *	 template: ....
	 *   strategy:
	 *     rollingUpdate:
	 *       maxSurge: 25%
	 *       maxUnavailable: 25%
	 *     type: RollingUpdate
	 */

	deployment := NewDefaultDeployment(namespace)

	annotations := map[string]string{}
	commonres.PropagateAnnotations(hawtioPlugin, annotations, log)

	labels, err := deploymentLabels(buildVariables)
	if err != nil {
		return nil, err
	}
	commonres.PropagateLabels(hawtioPlugin, labels, log)

	deployment.SetAnnotations(annotations)
	deployment.SetLabels(labels)
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				commonres.LabelAppKey:      commonres.LabelAppValue,
				commonres.LabelResourceKey: PluginName,
			},
		},
		Template: pts,
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		},
	}
	return deployment, nil
}

/**
 * Creates a new pod template
 */
func newPodTemplateSpec(hawtioPlugin *hawtiopluginv1.HawtioPlugin, buildVariables util.BuildVariables, log logr.Logger) (corev1.PodTemplateSpec, error) {
	log.V(util.DebugLogLevel).Info("New Pod Template Spec")

	pluginContainer := newContainer(hawtioPlugin, buildVariables, log)

	labels, err := deploymentLabels(buildVariables)
	if err != nil {
		return corev1.PodTemplateSpec{}, err
	}
	commonres.PropagateLabels(hawtioPlugin, labels, log)

	annotations := map[string]string{}
	commonres.PropagateAnnotations(hawtioPlugin, annotations, log)

	/*
	 *   template:
	 *     metadata:
	 *       labels:
	 *         app: hawtio
	 *         app.kubernetes.io/component: hawtio-online-console-plugin
	 *         deployment: hawtio-online-console-plugin
	 *         app.kubernetes.io/instance: hawtio-online-console-plugin
	 *         app.kubernetes.io/part-of: hawtio
	 *    spec:
	 *       containers: ....
	 *       volumes: ....
	 */

	volumes := newVolumes(hawtioPlugin, log)

	pod := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			// Used to trigger a rollout deployment if config changed,
			// similarly to `kubectl rollout restart`
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				pluginContainer,
			},
			ServiceAccountName: pluginres.ServiceAccountName,
			Volumes:            volumes,
		},
	}

	log.V(util.DebugLogLevel).Info(fmt.Sprintf("PodTemplateSpec: %s", util.ToJSONString(pod)))

	return pod, err
}

func newContainer(hawtioPlugin *hawtiopluginv1.HawtioPlugin, bv util.BuildVariables, log logr.Logger) corev1.Container {
	/*
	 * containers:
	 * - image: quay.io/phantomjinx/hawtio-online-console-plugin:0.9.0
	 *   imagePullPolicy: IfNotPresent
	 *   name: hawtio-online-console-plugin
	 *   ports:
	 *   - containerPort: 9443
	 *     name: nginx
	 *     protocol: TCP
	 *   resources: {}
	 */
	pluginVersion := bv.GetPluginVersion()
	log.V(util.DebugLogLevel).Info(fmt.Sprintf("Using Hawtio Plugin Image Version: %s", pluginVersion))

	/*
	 * readinessProbe:
	 *   failureThreshold: 3
	 *   httpGet:
	 *     port: nginx
	 *     path: /
	 *     scheme: HTTPS
	 *   initialDelaySeconds: 5
	 *   periodSeconds: 30
	 *   successThreshold: 1
	 *   timeoutSeconds: 1
	 */
	var readinessPeriodValue int32 = commonres.OnlineReadinessPeriodValue
	if hawtioPlugin.Spec.HealthChecks.PluginReadinessPeriod != nil {
		readinessPeriodValue = *hawtioPlugin.Spec.HealthChecks.PluginReadinessPeriod
	}

	/*
	 * livenessProbe:
	 *   failureThreshold: 3
	 *   httpGet:
	 *     port: nginx
	 *     path: /
	 *     scheme: HTTPS
	 *   periodSeconds: 120
	 *   successThreshold: 1
	 *   timeoutSeconds: 1
	 */
	log.V(util.DebugLogLevel).Info(fmt.Sprintf("Plugin Container protocols %s", util.ToJSONString(commonres.SSLConnect)))

	var livenessPeriodValue int32 = commonres.OnlineLivenessPeriodValue
	if hawtioPlugin.Spec.HealthChecks.PluginLivenessPeriod != nil {
		livenessPeriodValue = *hawtioPlugin.Spec.HealthChecks.PluginLivenessPeriod
	}

	/*
	 * volumeMounts: ...
	 */
	volumeMounts := newVolumeMounts(hawtioPlugin, bv, log)

	container := corev1.Container{
		Name:            hawtioPlugin.Name + "-plugin-container",
		Image:           bv.ImageURI(bv.GetPluginImageRepo(), pluginVersion),
		ImagePullPolicy: commonres.GetImagePullPolicy(),
		Ports: []corev1.ContainerPort{
			{
				Name:          commonres.ContainerPortName,
				ContainerPort: deploymentSSLPort,
				Protocol:      "TCP",
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port:   intstr.FromString(commonres.ContainerPortName),
					Path:   "/",
					Scheme: commonres.SSLConnect.Protocol,
				},
			},
			PeriodSeconds:  livenessPeriodValue,
			TimeoutSeconds: 1,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port:   intstr.FromString(commonres.ContainerPortName),
					Path:   "/",
					Scheme: commonres.SSLConnect.Protocol,
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       readinessPeriodValue,
			TimeoutSeconds:      1,
		},
		Resources:    hawtioPlugin.Spec.Resources,
		VolumeMounts: volumeMounts,
	}

	return container
}

func newVolumeMounts(hawtioPlugin *hawtiopluginv1.HawtioPlugin, buildVariables util.BuildVariables, log logr.Logger) []corev1.VolumeMount {
	/*
	 * - mountPath: /etc/tls/private/serving
	 *   name: hawtio-serving
	 *   readOnly: true
	 */

	volumeMounts := []corev1.VolumeMount{}

	/*
	 * The serving-certificate volume
	 */
	log.V(util.DebugLogLevel).Info(fmt.Sprintf("Adding volume mount %s at %s", serviceSigningSecretVolumeName, commonres.ServiceSigningSecretVolumeMountPath))
	volumeMount := commonres.NewVolumeMount(serviceSigningSecretVolumeName, commonres.ServiceSigningSecretVolumeMountPath, "")
	volumeMounts = append(volumeMounts, volumeMount)

	log.V(util.DebugLogLevel).Info(fmt.Sprintf("New VolumeMounts %s", util.ToJSONString(volumeMounts)))
	return volumeMounts
}

func newVolumes(hawtioPlugin *hawtiopluginv1.HawtioPlugin, log logr.Logger) []corev1.Volume {
	/*
	 * volumes:
	 *   - name: hawtio-serving
	 *     secret:
	 *       defaultMode: 420
	 *       secretName: hawtio-online-console-plugin-serving
	 */

	log.V(util.DebugLogLevel).Info("Creating new volumes")

	var volumes []corev1.Volume

	secretName := commonres.ServingSecretName(serviceName)
	log.V(util.DebugLogLevel).Info("Adding secret volume for serving certificate %s at %s", secretName, serviceSigningSecretVolumeName)
	volume := commonres.NewSecretVolume(secretName, serviceSigningSecretVolumeName)
	volumes = append(volumes, volume)

	return volumes
}
