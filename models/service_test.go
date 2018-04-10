package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1beta1"
	autoscalingV1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/prometheus"
	"github.com/prometheus/common/model"
)

func TestServiceDetailParsing(t *testing.T) {
	assert := assert.New(t)

	service := Service{}
	service.Name = "service"
	service.Namespace = Namespace{"namespace"}
	service.SetServiceDetails(fakeServiceDetails(), fakeIstioDetails(), fakePrometheusDetails())

	// Kubernetes Details
	assert.Equal(service.Name, "service")
	assert.Equal(service.Namespace.Name, "namespace")
	assert.Equal(service.Type, "ClusterIP")
	assert.Equal(service.Ip, "fromservice")
	assert.Equal(service.Labels, map[string]string{"label1": "labelName1", "label2": "labelName2"})
	assert.Equal(service.Ports, Ports{
		Port{Name: "http", Protocol: "TCP", Port: 3001},
		Port{Name: "http", Protocol: "TCP", Port: 3000}})
	assert.Equal(service.Endpoints, Endpoints{
		Endpoint{
			Addresses: Addresses{
				Address{Kind: "Pod", Name: "recommendation-v1", IP: "172.17.0.9"},
				Address{Kind: "Pod", Name: "recommendation-v2", IP: "172.17.0.8"},
			},
			Ports: Ports{
				Port{Name: "http", Protocol: "TCP", Port: 3001},
				Port{Name: "http", Protocol: "TCP", Port: 3000},
			}}})

	assert.Equal(*service.Deployments[0], Deployment{
		Name:                "reviews-v1",
		Labels:              map[string]string{"apps": "reviews", "version": "v1"},
		CreatedAt:           "2018-03-08T17:44:00+03:00",
		Replicas:            3,
		AvailableReplicas:   1,
		UnavailableReplicas: 2,
		Autoscaler: Autoscaler{
			Name:                            "reviews-v1",
			Labels:                          map[string]string{"apps": "reviews", "version": "v1"},
			CreatedAt:                       "2018-03-08T17:44:00+03:00",
			MinReplicas:                     1,
			MaxReplicas:                     10,
			TargetCPUUtilizationPercentage:  50,
			CurrentReplicas:                 3,
			DesiredReplicas:                 4,
			ObservedGeneration:              50,
			CurrentCPUUtilizationPercentage: 70}})

	assert.Equal(*service.Deployments[1], Deployment{
		Name:                "reviews-v2",
		Labels:              map[string]string{"apps": "reviews", "version": "v2"},
		CreatedAt:           "2018-03-08T17:45:00+03:00",
		Replicas:            3,
		AvailableReplicas:   3,
		UnavailableReplicas: 0,
		Autoscaler: Autoscaler{
			Name:                            "reviews-v2",
			Labels:                          map[string]string{"apps": "reviews", "version": "v2"},
			CreatedAt:                       "2018-03-08T17:45:00+03:00",
			MinReplicas:                     1,
			MaxReplicas:                     10,
			TargetCPUUtilizationPercentage:  50,
			CurrentReplicas:                 3,
			DesiredReplicas:                 2,
			ObservedGeneration:              50,
			CurrentCPUUtilizationPercentage: 30}})

	// Istio Details
	assert.Equal(service.RouteRules, RouteRules{
		RouteRule{
			Destination: map[string]string{
				"name":      "reviews",
				"namespace": "tutorial"},
			Precedence: 1,
			Route: map[string]map[string]string{
				"labels": {
					"name":      "version",
					"namespace": "v1"}},
			HttpFault: map[string]map[string]string{
				"abort": {
					"percent":    "50",
					"httpStatus": "503",
				},
			}},
		RouteRule{
			Destination: map[string]string{
				"name":      "reviews",
				"namespace": "tutorial"},
			Precedence: 1,
			Route: map[string]map[string]string{
				"labels": {
					"name":      "version",
					"namespace": "v3"}}}})

	assert.Equal(service.DestinationPolicies, DestinationPolicies{
		DestinationPolicy{
			Source: map[string]string{
				"name": "recommendation"},
			Destination: map[string]string{
				"name":      "reviews",
				"namespace": "tutorial"},
			LoadBalancing: map[string]string{
				"name": "RANDOM"},
		},
		DestinationPolicy{
			Destination: map[string]interface{}{
				"name":      "reviews",
				"namespace": "tutorial",
				"labels": map[string]string{
					"version": "v2"}},
			CircuitBreaker: map[string]interface{}{
				"simpleCb": map[string]interface{}{
					"maxConnections":               1,
					"httpMaxPendingRequests":       1,
					"sleepWindow":                  "2m",
					"httpDetectionInterval":        "1s",
					"httpMaxEjectionPercent":       100,
					"httpConsecutiveErrors":        1,
					"httpMaxRequestsPerConnection": 1,
				}},
		}})

	// Prometheus Client
	assert.Equal(service.Dependencies, map[string][]string{
		"v1": {"unknown", "/products", "/reviews"},
		"v2": {"/catalog", "/shares"}})
}

func fakeServiceDetails() *kubernetes.ServiceDetails {
	service := &v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "Name",
			Namespace: "Namespace",
			Labels: map[string]string{
				"label1": "labelName1",
				"label2": "labelName2"}},
		Spec: v1.ServiceSpec{
			ClusterIP: "fromservice",
			Type:      "ClusterIP",
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     3001},
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     3000}}}}

	endpoints := &v1.Endpoints{
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: "172.17.0.9",
						TargetRef: &v1.ObjectReference{
							Kind: "Pod",
							Name: "recommendation-v1"}},
					{
						IP: "172.17.0.8",
						TargetRef: &v1.ObjectReference{
							Kind: "Pod",
							Name: "recommendation-v2"}},
				},
				Ports: []v1.EndpointPort{
					{Name: "http", Protocol: "TCP", Port: 3001},
					{Name: "http", Protocol: "TCP", Port: 3000},
				}}}}

	t1, _ := time.Parse(time.RFC822Z, "08 Mar 18 17:44 +0300")
	t2, _ := time.Parse(time.RFC822Z, "08 Mar 18 17:45 +0300")
	deployments := &v1beta1.DeploymentList{
		Items: []v1beta1.Deployment{
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:              "reviews-v1",
					CreationTimestamp: meta_v1.NewTime(t1),
					Labels:            map[string]string{"apps": "reviews", "version": "v1"}},
				Status: v1beta1.DeploymentStatus{
					Replicas:            3,
					AvailableReplicas:   1,
					UnavailableReplicas: 2}},
			v1beta1.Deployment{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:              "reviews-v2",
					CreationTimestamp: meta_v1.NewTime(t2),
					Labels:            map[string]string{"apps": "reviews", "version": "v2"}},
				Status: v1beta1.DeploymentStatus{
					Replicas:            3,
					AvailableReplicas:   3,
					UnavailableReplicas: 0}}}}

	autoscalers := &autoscalingV1.HorizontalPodAutoscalerList{
		Items: []autoscalingV1.HorizontalPodAutoscaler{
			autoscalingV1.HorizontalPodAutoscaler{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:              "reviews-v1",
					Labels:            map[string]string{"apps": "reviews", "version": "v1"},
					CreationTimestamp: meta_v1.NewTime(t1)},
				Spec: autoscalingV1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingV1.CrossVersionObjectReference{
						Name: "reviews-v1"},
					MinReplicas:                    &[]int32{1}[0],
					MaxReplicas:                    10,
					TargetCPUUtilizationPercentage: &[]int32{50}[0]},
				Status: autoscalingV1.HorizontalPodAutoscalerStatus{
					ObservedGeneration:              &[]int64{50}[0],
					CurrentReplicas:                 3,
					DesiredReplicas:                 4,
					CurrentCPUUtilizationPercentage: &[]int32{70}[0]}},
			autoscalingV1.HorizontalPodAutoscaler{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:              "reviews-v2",
					Labels:            map[string]string{"apps": "reviews", "version": "v2"},
					CreationTimestamp: meta_v1.NewTime(t2)},
				Spec: autoscalingV1.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingV1.CrossVersionObjectReference{
						Name: "reviews-v2"},
					MinReplicas:                    &[]int32{1}[0],
					MaxReplicas:                    10,
					TargetCPUUtilizationPercentage: &[]int32{50}[0]},
				Status: autoscalingV1.HorizontalPodAutoscalerStatus{
					ObservedGeneration:              &[]int64{50}[0],
					CurrentReplicas:                 3,
					DesiredReplicas:                 2,
					CurrentCPUUtilizationPercentage: &[]int32{30}[0]}}}}

	return &kubernetes.ServiceDetails{service, endpoints, deployments, autoscalers}
}

func fakeIstioDetails() *kubernetes.IstioDetails {
	route1 := kubernetes.MockIstioObject{
		Spec: map[string]interface{}{
			"destination": map[string]string{
				"name":      "reviews",
				"namespace": "tutorial"},
			"precedence": 1,
			"route": map[string]map[string]string{
				"labels": map[string]string{
					"name":      "version",
					"namespace": "v1"}},
			"httpFault": map[string]map[string]string{
				"abort": map[string]string{
					"percent":    "50",
					"httpStatus": "503",
				}}},
	}
	route2 := kubernetes.MockIstioObject{
		Spec: map[string]interface{}{
			"destination": map[string]string{
				"name":      "reviews",
				"namespace": "tutorial"},
			"precedence": 1,
			"route": map[string]map[string]string{
				"labels": map[string]string{
					"name":      "version",
					"namespace": "v3"}}},
	}
	routes := []kubernetes.IstioObject{&route1, &route2}
	policy1 := kubernetes.MockIstioObject{
		Spec: map[string]interface{}{
			"source": map[string]string{
				"name": "recommendation",
			},
			"destination": map[string]string{
				"name":      "reviews",
				"namespace": "tutorial",
			},
			"loadBalancing": map[string]string{
				"name": "RANDOM",
			},
		},
	}
	policy2 := kubernetes.MockIstioObject{
		Spec: map[string]interface{}{
			"destination": map[string]interface{}{
				"name":      "reviews",
				"namespace": "tutorial",
				"labels": map[string]string{
					"version": "v2",
				},
			},
			"circuitBreaker": map[string]interface{}{
				"simpleCb": map[string]interface{}{
					"maxConnections":               1,
					"httpMaxPendingRequests":       1,
					"sleepWindow":                  "2m",
					"httpDetectionInterval":        "1s",
					"httpMaxEjectionPercent":       100,
					"httpConsecutiveErrors":        1,
					"httpMaxRequestsPerConnection": 1,
				},
			},
		},
	}
	policies := []kubernetes.IstioObject{&policy1, &policy2}
	return &kubernetes.IstioDetails{routes, policies}
}

func fakePrometheusDetails() map[string][]string {
	return map[string][]string{
		"v1": []string{"unknown", "/products", "/reviews"},
		"v2": []string{"/catalog", "/shares"}}
}

func TestServiceListParsing(t *testing.T) {
	assert := assert.New(t)

	fakeConfig()
	serviceList := ServiceList{}
	serviceList.Namespace = Namespace{"namespace"}
	serviceList.SetServiceList(fakeServiceList())

	assert.Equal("namespace", serviceList.Namespace.Name)
	assert.Equal(serviceList.Services, []ServiceOverview{
		ServiceOverview{
			Name:                "reviews",
			Replicas:            int32(5),
			AvailableReplicas:   int32(3),
			UnavailableReplicas: int32(2)},
		ServiceOverview{
			Name:                "httpbin",
			Replicas:            int32(1),
			AvailableReplicas:   int32(1),
			UnavailableReplicas: int32(0)}})
}

func TestRequestCountersProcessing(t *testing.T) {
	assert := assert.New(t)

	serviceList := ServiceList{}
	serviceList.Namespace = Namespace{"namespace"}
	serviceList.SetServiceList(fakeServiceList())
	serviceList.ProcessRequestCounters(*fakeRequestCounters())

	reviewsOverview := serviceList.Services[0]
	assert.Equal(model.SampleValue(6.5), reviewsOverview.RequestCount)
	assert.Equal(model.SampleValue(1.5), reviewsOverview.RequestErrorCount)
	assert.Equal(model.SampleValue(1.5/6.5), reviewsOverview.ErrorRate)

	httpbinOverview := serviceList.Services[1]
	assert.Equal(model.SampleValue(20.5), httpbinOverview.RequestCount)
	assert.Equal(model.SampleValue(1.5), httpbinOverview.RequestErrorCount)
	assert.Equal(model.SampleValue(1.5/20.5), httpbinOverview.ErrorRate)
}

func fakeConfig() {
	conf := config.NewConfig()
	config.Set(conf)
}

func fakeServiceList() *kubernetes.ServiceList {
	t1, _ := time.Parse(time.RFC822Z, "08 Mar 18 17:44 +0300")
	t2, _ := time.Parse(time.RFC822Z, "08 Mar 18 17:45 +0300")

	return &kubernetes.ServiceList{
		Services: &v1.ServiceList{
			Items: []v1.Service{
				v1.Service{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "reviews",
						Namespace: "tutorial",
						Labels: map[string]string{
							"app":     "reviews",
							"version": "v1"}},
					Spec: v1.ServiceSpec{
						ClusterIP: "fromservice",
						Type:      "ClusterIP",
						Selector:  map[string]string{"app": "reviews"},
						Ports: []v1.ServicePort{
							{
								Name:     "http",
								Protocol: "TCP",
								Port:     3001},
							{
								Name:     "http",
								Protocol: "TCP",
								Port:     3000}}}},
				v1.Service{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "httpbin",
						Namespace: "tutorial",
						Labels: map[string]string{
							"app":     "httpbin",
							"version": "v1"}},
					Spec: v1.ServiceSpec{
						ClusterIP: "fromservice",
						Type:      "ClusterIP",
						Selector:  map[string]string{"app": "httpbin"},
						Ports: []v1.ServicePort{
							{
								Name:     "http",
								Protocol: "TCP",
								Port:     3001},
							{
								Name:     "http",
								Protocol: "TCP",
								Port:     3000}}}},
			}},
		Deployments: &v1beta1.DeploymentList{
			Items: []v1beta1.Deployment{
				v1beta1.Deployment{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "reviews-v1",
						CreationTimestamp: meta_v1.NewTime(t1),
						Labels:            map[string]string{"app": "reviews", "version": "v1"}},
					Status: v1beta1.DeploymentStatus{
						Replicas:            3,
						AvailableReplicas:   2,
						UnavailableReplicas: 1}},
				v1beta1.Deployment{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "reviews-v2",
						CreationTimestamp: meta_v1.NewTime(t1),
						Labels:            map[string]string{"app": "reviews", "version": "v2"}},
					Status: v1beta1.DeploymentStatus{
						Replicas:            2,
						AvailableReplicas:   1,
						UnavailableReplicas: 1}},
				v1beta1.Deployment{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:              "httpbin-v1",
						CreationTimestamp: meta_v1.NewTime(t2),
						Labels:            map[string]string{"app": "httpbin", "version": "v1"}},
					Status: v1beta1.DeploymentStatus{
						Replicas:            1,
						AvailableReplicas:   1,
						UnavailableReplicas: 0}}}}}
}

func fakeRequestCounters() *prometheus.MetricsVector {
	t1 := model.Now()

	return &prometheus.MetricsVector{
		Vector: model.Vector{
			&model.Sample{
				Metric: model.Metric{
					"destination_service": "reviews.tutorial.svc.cluster.local",
					"source_service":      "httpbin.tutorial.svc.cluster.local",
					"response_code":       "200",
				},
				Value:     model.SampleValue(5),
				Timestamp: t1,
			},
			&model.Sample{
				Metric: model.Metric{
					"destination_service": "httpbin.tutorial.svc.cluster.local",
					"source_service":      "unknown",
					"response_code":       "200",
				},
				Value:     model.SampleValue(14),
				Timestamp: t1,
			},
			&model.Sample{
				Metric: model.Metric{
					"source_service":      "httpbin.tutorial.svc.cluster.local",
					"destination_service": "unknown",
					"response_code":       "400",
				},
				Value:     model.SampleValue(1.5),
				Timestamp: t1,
			},
			&model.Sample{
				Metric: model.Metric{
					"source_service":      "reviews.tutorial.svc.cluster.local",
					"destination_service": "unknown",
					"response_code":       "500",
				},
				Value:     model.SampleValue(1.5),
				Timestamp: t1,
			},
		},
	}
}