package service

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ut "github.com/zdnscloud/cement/unittest"
	"github.com/zdnscloud/cluster-agent/service/testutil"
)

func TestMonitorHandleIngressEvent(t *testing.T) {
	cache := testutil.NewMockCache()
	monitor := newServiceMonitor(cache)

	newSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "vanguard", Namespace: "default"},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "vanguard"}},
	}
	cache.SetGetResult(&corev1.PodList{Items: nil})
	monitor.OnNewService(newSvc)

	innerServices := monitor.GetInnerServices()
	ut.Equal(t, len(innerServices), 1)

	httpIng := &Ingress{
		Name: "vanguard",
		Rules: []IngressRule{
			IngressRule{
				Host:     "www.knet.cn",
				Protocol: IngressProtocolHTTP,
				Paths: []IngressPath{
					IngressPath{
						Path:        "/v1",
						ServiceName: "vanguard",
						ServicePort: 8000,
					},
				},
			},
		},
	}
	monitor.addIngress(httpIng)

	innerServices = monitor.GetInnerServices()
	ut.Equal(t, len(innerServices), 0)

	outerServices := monitor.GetOuterServices()
	ut.Equal(t, len(outerServices), 1)

	ut.Equal(t, outerServices[0].EntryPoint, "http://www.knet.cn")
	ut.Equal(t, outerServices[0].Services["/v1"].Name, "vanguard")

	udpIng := &Ingress{
		Name: "vanguard",
		Rules: []IngressRule{
			IngressRule{
				Port:     5553,
				Protocol: IngressProtocolUDP,
				Paths: []IngressPath{
					IngressPath{
						ServiceName: "vanguard",
						ServicePort: 8000,
					},
				},
			},
		},
	}
	monitor.addIngress(udpIng)

	outerServices = monitor.GetOuterServices()
	ut.Equal(t, len(outerServices), 2)

	ut.Equal(t, outerServices[1].EntryPoint, "udp:5553")
	ut.Equal(t, outerServices[1].Services[""].Name, "vanguard")

	monitor.OnDeleteTransportLayerIngress(udpIng)
	outerServices = monitor.GetOuterServices()
	ut.Equal(t, len(outerServices), 1)
	ut.Equal(t, outerServices[0].EntryPoint, "http://www.knet.cn")
	ut.Equal(t, outerServices[0].Services["/v1"].Name, "vanguard")

	monitor.OnDeleteTransportLayerIngress(httpIng)
	outerServices = monitor.GetOuterServices()
	ut.Equal(t, len(outerServices), 0)
	innerServices = monitor.GetInnerServices()
	ut.Equal(t, len(innerServices), 1)
	ut.Equal(t, len(outerServices), 0)
}
