package network

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ut "github.com/zdnscloud/cement/unittest"
)

const (
	OpCreate = "create"
	OpDelete = "delete"
	OpUpdate = "update"
)

func TestNodeNetwork(t *testing.T) {
	nc := newNetworkCache()
	testCases := []struct {
		Operation            string
		Name                 string
		PodCIDR              string
		Address              string
		ExpectNodeNetworkCnt int
		ExpectPodNetworkCnt  int
		ExpectNodeAddress    string
		ExpectPodCIDR        string
	}{
		{
			Operation:            OpCreate,
			Name:                 "master",
			PodCIDR:              "",
			Address:              "192.168.1.126",
			ExpectNodeNetworkCnt: 1,
			ExpectPodNetworkCnt:  0,
			ExpectNodeAddress:    "192.168.1.126",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpCreate,
			Name:                 "master",
			PodCIDR:              "",
			Address:              "192.168.1.125",
			ExpectNodeNetworkCnt: 1,
			ExpectPodNetworkCnt:  0,
			ExpectNodeAddress:    "192.168.1.126",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpCreate,
			Name:                 "worker1",
			PodCIDR:              "",
			Address:              "192.168.1.127",
			ExpectNodeNetworkCnt: 2,
			ExpectPodNetworkCnt:  0,
			ExpectNodeAddress:    "192.168.1.127",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpCreate,
			Name:                 "worker2",
			PodCIDR:              "",
			Address:              "192.168.1.128",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  0,
			ExpectNodeAddress:    "192.168.1.128",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpUpdate,
			Name:                 "master",
			PodCIDR:              "10.42.1.0/24",
			Address:              "192.168.1.126",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  1,
			ExpectNodeAddress:    "192.168.1.126",
			ExpectPodCIDR:        "10.42.1.0/24",
		},
		{
			Operation:            OpUpdate,
			Name:                 "nonexist",
			PodCIDR:              "10.42.2.0/24",
			Address:              "192.168.1.127",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  1,
			ExpectNodeAddress:    "",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpUpdate,
			Name:                 "worker1",
			PodCIDR:              "10.42.2.0/24",
			Address:              "192.168.1.127",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  2,
			ExpectNodeAddress:    "192.168.1.127",
			ExpectPodCIDR:        "10.42.2.0/24",
		},
		{
			Operation:            OpUpdate,
			Name:                 "worker2",
			PodCIDR:              "10.42.3.0/24",
			Address:              "192.168.1.128",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  3,
			ExpectNodeAddress:    "192.168.1.128",
			ExpectPodCIDR:        "10.42.3.0/24",
		},
		{
			Operation:            OpUpdate,
			Name:                 "worker2",
			PodCIDR:              "10.42.4.0/24",
			Address:              "192.168.1.128",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  3,
			ExpectNodeAddress:    "192.168.1.128",
			ExpectPodCIDR:        "10.42.4.0/24",
		},
		{
			Operation:            OpDelete,
			Name:                 "nonexist",
			PodCIDR:              "10.42.2.0/24",
			Address:              "192.168.1.127",
			ExpectNodeNetworkCnt: 3,
			ExpectPodNetworkCnt:  3,
			ExpectNodeAddress:    "",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpDelete,
			Name:                 "worker1",
			PodCIDR:              "10.42.2.0/24",
			Address:              "192.168.1.127",
			ExpectNodeNetworkCnt: 2,
			ExpectPodNetworkCnt:  2,
			ExpectNodeAddress:    "",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpDelete,
			Name:                 "worker2",
			PodCIDR:              "10.42.4.0/24",
			Address:              "192.168.1.128",
			ExpectNodeNetworkCnt: 1,
			ExpectPodNetworkCnt:  1,
			ExpectNodeAddress:    "",
			ExpectPodCIDR:        "",
		},
		{
			Operation:            OpDelete,
			Name:                 "master",
			PodCIDR:              "10.42.1.0/24",
			Address:              "192.168.1.126",
			ExpectNodeNetworkCnt: 0,
			ExpectPodNetworkCnt:  0,
			ExpectNodeAddress:    "",
			ExpectPodCIDR:        "",
		},
	}

	for _, testCase := range testCases {
		node := newNode(testCase.Name, testCase.PodCIDR, testCase.Address)
		switch testCase.Operation {
		case OpCreate:
			nc.OnNewNode(node)
		case OpDelete:
			nc.OnDeleteNode(node)
		case OpUpdate:
			nc.OnUpdateNode(node)
		}
		nodeNetworkSlice := nc.GetNodeNetworks()
		ut.Equal(t, len(nodeNetworkSlice), testCase.ExpectNodeNetworkCnt)
		podNetworkSlice := nc.GetPodNetworks()
		ut.Equal(t, len(podNetworkSlice), testCase.ExpectPodNetworkCnt)
		nodeNetwork, ok := nc.nodeNetworks[node.Name]
		ut.Equal(t, ok, testCase.ExpectNodeAddress != "")
		if ok {
			ut.Equal(t, nodeNetwork.IP, testCase.ExpectNodeAddress)
		}

		podNetwork, ok := nc.podNetworks[node.Name]
		ut.Equal(t, ok, testCase.ExpectPodCIDR != "")
		if ok {
			ut.Equal(t, podNetwork.PodCIDR, testCase.ExpectPodCIDR)
		}
	}
}

func newNode(name, podCIDR, address string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{PodCIDR: podCIDR},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				corev1.NodeAddress{
					Type:    corev1.NodeInternalIP,
					Address: address,
				},
				corev1.NodeAddress{
					Type:    corev1.NodeHostName,
					Address: name,
				},
			},
		},
	}
}
