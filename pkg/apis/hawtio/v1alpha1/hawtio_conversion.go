package v1alpha1

import (
	"errors"

	hawtiov1 "github.com/hawtio/hawtio-operator/pkg/apis/hawtio/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this Hawtio to the Hub version (v1).
func (src *Hawtio) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*hawtiov1.Hawtio)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	deployType := string(src.Spec.Type)
	dst.Spec.Type = hawtiov1.HawtioDeploymentType(deployType)
	dst.Spec.Replicas = src.Spec.Replicas

	// src.Spec.Version is dropped

	dst.Spec.MetadataPropagation = hawtiov1.HawtioMetadataPropagation{
		Annotations: src.Spec.MetadataPropagation.Annotations,
		Labels:      src.Spec.MetadataPropagation.Labels,
	}

	dst.Spec.RouteHostName = src.Spec.RouteHostName

	dst.Spec.Route = hawtiov1.HawtioRoute{
		CertSecret: src.Spec.Route.CertSecret,
		CaCert:     src.Spec.Route.CaCert,
	}

	dst.Spec.ExternalRoutes = src.Spec.ExternalRoutes

	dst.Spec.Auth = hawtiov1.HawtioAuth{
		ClientCertCommonName:       src.Spec.Auth.ClientCertCommonName,
		ClientCertExpirationDate:   src.Spec.Auth.ClientCertExpirationDate,
		ClientCertCheckSchedule:    src.Spec.Auth.ClientCertCheckSchedule,
		ClientCertExpirationPeriod: src.Spec.Auth.ClientCertExpirationPeriod,
	}

	dst.Spec.Nginx = hawtiov1.HawtioNginx{
		ClientBodyBufferSize:       src.Spec.Nginx.ClientBodyBufferSize,
		ProxyBuffers:               src.Spec.Nginx.ProxyBuffers,
		SubrequestOutputBufferSize: src.Spec.Nginx.SubrequestOutputBufferSize,
	}

	dst.Spec.RBAC = hawtiov1.HawtioRBAC{
		ConfigMap:           src.Spec.RBAC.ConfigMap,
		DisableRBACRegistry: src.Spec.RBAC.DisableRBACRegistry,
	}

	dst.Spec.Resources = src.Spec.Resources

	dst.Spec.Config = hawtiov1.HawtioConfig{}

	dst.Spec.Config.About.Title = src.Spec.Config.About.Title

	if src.Spec.Config.About.ProductInfos != nil && len(src.Spec.Config.About.ProductInfos) > 0 {
		for _, spi := range src.Spec.Config.About.ProductInfos {
			dpi := hawtiov1.HawtioProductInfo{
				Name:  spi.Name,
				Value: spi.Value,
			}
			dst.Spec.Config.About.ProductInfos = append(dst.Spec.Config.About.ProductInfos, dpi)
		}
	}

	dst.Spec.Config.About.AdditionalInfo = src.Spec.Config.About.AdditionalInfo
	dst.Spec.Config.About.Copyright = src.Spec.Config.About.Copyright
	dst.Spec.Config.About.ImgSrc = src.Spec.Config.About.ImgSrc

	dst.Spec.Config.Branding = hawtiov1.HawtioBranding{
		AppName:    src.Spec.Config.Branding.AppName,
		AppLogoURL: src.Spec.Config.Branding.AppLogoURL,
		CSS:        src.Spec.Config.Branding.CSS,
		Favicon:    src.Spec.Config.Branding.Favicon,
	}

	dst.Spec.Config.Online = hawtiov1.HawtioOnline{}
	dst.Spec.Config.Online.ProjectSelector = src.Spec.Config.Online.ProjectSelector
	dst.Spec.Config.Online.ConsoleLink = hawtiov1.HawtioConsoleLink{
		Text:              src.Spec.Config.Online.ConsoleLink.Text,
		Section:           src.Spec.Config.Online.ConsoleLink.Section,
		ImageRelativePath: src.Spec.Config.Online.ConsoleLink.ImageRelativePath,
	}

	dst.Spec.Config.DisabledRoutes = src.Spec.Config.DisabledRoutes

	// Hawtio Status
	phase := string(src.Status.Phase)
	dstPhase := hawtiov1.HawtioPhase(phase)
	dst.Status = hawtiov1.HawtioStatus{
		Image:    src.Status.Image,
		Phase:    dstPhase,
		URL:      src.Status.URL,
		Replicas: src.Status.Replicas,
		Selector: src.Status.Selector,
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1) to this version.
func (dst *Hawtio) ConvertFrom(srcRaw conversion.Hub) error {
	err := errors.New("Conversion from v1 to v1alpha1 is not supported")
	return err
}
