package common

import (
	commonapi "github.com/hawtio/hawtio-operator/pkg/apis/common"
)

// ServiceNameFromHawtio derives the name of the service
// from the Hawtio CR
func ServiceNameFromHawtio(hawtio commonapi.HawtioGeneral) string {
	metadataInfo := hawtio.GetMetadataInfo()
	return metadataInfo.Name
}

// ServingSecretName derives the name of the serving secret
// from the given service name
func ServingSecretName(serviceName string) string {
	return serviceName + "-tls-serving"
}
