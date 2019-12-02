package services

import (
	"testing"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/models"
)

func TestPortMappingMatch(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	pmc := PortMappingChecker{
		Service:     getService(9080, "http"),
		Deployments: getDeployment(9080),
		Pods:        getPods(true),
	}

	validations, valid := pmc.Check()
	assert.True(valid)
	assert.Empty(validations)
}

func TestTargetPortMappingMatch(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	service := getService(9080, "http")
	service.Spec.Ports[0].TargetPort = intstr.FromInt(8080)

	/*
		// If this is a string, it will be looked up as a named port in the
		// target Pod's container ports. If this is not specified, the value
		// of the 'port' field is used (an identity map).
		// This field is ignored for services with clusterIP=None, and should be
		// omitted or set equal to the 'port' field.

	*/

	pmc := PortMappingChecker{
		Service:     service,
		Deployments: getDeployment(8080),
		Pods:        getPods(true),
	}

	validations, valid := pmc.Check()
	assert.True(valid)
	assert.Empty(validations)

	// Now check with named port only
	service.Spec.Ports[0].TargetPort = intstr.FromString("http-container")

	validations, valid = pmc.Check()
	assert.True(valid)
	assert.Empty(validations)
}

func TestPortMappingMismatch(t *testing.T) {
	// As per KIALI-2454
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	pmc := PortMappingChecker{
		Service:     getService(9080, "http"),
		Deployments: getDeployment(8080),
		Pods:        getPods(true),
	}

	validations, valid := pmc.Check()
	assert.False(valid)
	assert.NotEmpty(validations)
	assert.Equal(models.CheckMessage("service.deployment.port.mismatch"), validations[0].Message)
	assert.Equal("spec/ports[0]", validations[0].Path)
}

func TestServicePortNaming(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	pmc := PortMappingChecker{
		Service:     getService(9080, "http2foo"),
		Deployments: getDeployment(9080),
		Pods:        getPods(true),
	}

	validations, valid := pmc.Check()
	assert.False(valid)
	assert.NotEmpty(validations)
	assert.Equal(models.CheckMessage("port.name.mismatch"), validations[0].Message)
	assert.Equal("spec/ports[0]", validations[0].Path)
}

func TestServicePortNamingWithoutSidecar(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	pmc := PortMappingChecker{
		Service:     getService(9080, "http2foo"),
		Deployments: getDeployment(9080),
		Pods:        getPods(false),
	}

	validations, valid := pmc.Check()
	assert.True(valid)
	assert.Empty(validations)
}

func getService(servicePort int32, portName string) v1.Service {
	return v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "service1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: servicePort,
					Name: portName,
				},
			},
			Selector: map[string]string{
				"dep": "one",
			},
		},
	}
}

func getDeployment(containerPort int32) []apps_v1.Deployment {
	return []apps_v1.Deployment{
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Labels: map[string]string{
					"dep": "one",
				},
			},
			Spec: apps_v1.DeploymentSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							{
								Ports: []v1.ContainerPort{
									{
										Name:          "http-container",
										ContainerPort: containerPort,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getPods(withSidecar bool) []v1.Pod {
	conf := config.NewConfig()

	annotation := "sidecarless-annotation"
	if withSidecar {
		annotation = conf.ExternalServices.Istio.IstioSidecarAnnotation
	}

	return []v1.Pod{
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Labels: map[string]string{
					"dep": "one",
				},
				Annotations: map[string]string{
					annotation: "{\"version\":\"\",\"initContainers\":[\"istio-init\",\"enable-core-dump\"],\"containers\":[\"istio-proxy\"],\"volumes\":[\"istio-envoy\",\"istio-certs\"]}",
				},
			},
		},
	}
}
