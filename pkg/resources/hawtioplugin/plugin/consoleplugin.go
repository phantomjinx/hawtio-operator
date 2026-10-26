package plugin

import (
	"github.com/go-logr/logr"

	osv1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hawtiopluginv1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtioplugin/v1"
	commonres "github.com/hawtio/hawtio-operator/pkg/resources/common"
	pluginres "github.com/hawtio/hawtio-operator/pkg/resources/hawtioplugin"

	"github.com/hawtio/hawtio-operator/pkg/util"
)

// NewDefaultConsolePlugin create default basic ConsolePlugin
// ConsolePlugin is a cluster-level resource
func NewDefaultConsolePlugin() *osv1.ConsolePlugin {
	return &osv1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: PluginName,
		},
	}
}

func NewConsolePlugin(hawtioPlugin *hawtiopluginv1.HawtioPlugin, namespace string, buildVariables util.BuildVariables, log logr.Logger) (*osv1.ConsolePlugin, error) {
	log.V(util.DebugLogLevel).Info("Reconciling console plugin")

	//
	// apiVersion: console.openshift.io/v1
	// kind: ConsolePlugin
	// metadata:
	//   labels:
	//     app: hawtio
	//     plugin: hawtio-online-console-plugin
	//     app.kubernetes.io/component: hawtio-online-console-plugin
	//     app.kubernetes.io/instance: hawtio-online-console-plugin
	//     app.kubernetes.io/part-of: hawtio
	//   name: hawtio-online-console-plugin
	// spec:
	//   backend:
	//     service:
	//       basePath: /
	//       name: hawtio-online-console-plugin
	//       namespace: hawtio-dev
	//       port: 9443
	//     type: Service
	//   displayName: OpenShift Console Hawtio Online Plugin
	//   proxy:
	//   - alias: gateway
	//     authorization: UserToken
	//     endpoint:
	//       service:
	//         name: hawtio-online-console-plugin-gateway
	//         namespace: <operator namespace>
	//         port: 8443
	//       type: Service

	plugin := NewDefaultConsolePlugin()

	annotations := map[string]string{}
	commonres.PropagateAnnotations(hawtioPlugin, annotations, log)

	labels, err := pluginres.ResourceLabels(PluginName, buildVariables)
	if err != nil {
		return nil, err
	}
	commonres.PropagateLabels(hawtioPlugin, labels, log)

	plugin.SetAnnotations(annotations)
	plugin.SetLabels(labels)
	plugin.Spec = osv1.ConsolePluginSpec{
		Backend: osv1.ConsolePluginBackend{
			Service: &osv1.ConsolePluginService{
				Name:      serviceName,
				Namespace: namespace,
				Port:      int32(pluginSSLPort),
				BasePath:  "/",
			},
			Type: osv1.Service,
		},
		DisplayName: pluginDisplayName,
		Proxy: []osv1.ConsolePluginProxy{
			osv1.ConsolePluginProxy{
				Alias:         "gateway",
				Authorization: osv1.UserToken,
				Endpoint: osv1.ConsolePluginProxyEndpoint{
					Service: &osv1.ConsolePluginProxyServiceConfig{
						Name:      gatewayServiceName,
						Namespace: namespace,
						Port:      int32(gatewaySSLPort),
					},
					Type: osv1.ProxyTypeService,
				},
			},
		},
	}
	return plugin, nil
}
