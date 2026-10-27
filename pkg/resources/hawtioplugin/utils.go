package hawtioplugin

import (
	labelUtils "k8s.io/apimachinery/pkg/labels"

	commonres "github.com/hawtio/hawtio-operator/pkg/resources/common"
	"github.com/hawtio/hawtio-operator/pkg/util"
)

// ResourceLabels builds the necessary labels for a plugin resource
func ResourceLabels(name string, buildVariables util.BuildVariables) (map[string]string, error) {
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
	labels := commonres.LabelsForHawtio(name)
	labels["app.kubernetes.io/component"] = name
	labels["app.kubernetes.io/instance"] = name
	labels["app.kubernetes.io/part-of"] = commonres.LabelAppValue

	for nm, value := range additionalLabels {
		labels[nm] = value
	}

	return labels, nil
}
