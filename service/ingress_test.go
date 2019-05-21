package service

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
)

func TestIngressRemoveRule(t *testing.T) {
	cases := []struct {
		src           *Ingress
		protocol      IngressProtocol
		leftRuleCount int
	}{
		{
			&Ingress{
				name: "i1",
				rules: []IngressRule{
					IngressRule{
						protocol: IngressProtocolHTTP,
					},
				},
			},
			IngressProtocolHTTP,
			0,
		},
		{
			&Ingress{
				name: "i1",
				rules: []IngressRule{
					IngressRule{
						protocol: IngressProtocolHTTP,
					},
					IngressRule{
						protocol: IngressProtocolUDP,
					},
					IngressRule{
						protocol: IngressProtocolTCP,
					},
				},
			},
			IngressProtocolHTTP,
			2,
		},
		{
			&Ingress{
				name: "i1",
				rules: []IngressRule{
					IngressRule{
						protocol: IngressProtocolHTTP,
					},
					IngressRule{
						protocol: IngressProtocolUDP,
					},
					IngressRule{
						protocol: IngressProtocolTCP,
					},
				},
			},
			IngressProtocolUDP,
			1,
		},
		{
			&Ingress{
				name: "i1",
				rules: []IngressRule{
					IngressRule{
						protocol: IngressProtocolHTTP,
					},
					IngressRule{
						protocol: IngressProtocolUDP,
					},
				},
			},
			IngressProtocolTCP,
			1,
		},
	}

	for _, tc := range cases {
		ingressRemoveRules(tc.src, tc.protocol)
		ut.Equal(t, len(tc.src.rules), tc.leftRuleCount)
	}
}
