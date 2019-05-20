package service

import (
	"fmt"
	"strconv"
	"strings"
)

type IngressProtocol string

const (
	IngressProtocolUDP  IngressProtocol = "udp"
	IngressProtocolTCP  IngressProtocol = "tcp"
	IngressProtocolHTTP IngressProtocol = "http"
)

const (
	NginxIngressNamespace = "ingress-nginx"
	NginxUDPConfigMapName = "udp-services"
	NginxTCPConfigMapName = "tcp-services"
)

func configMapToIngresses(configs map[string]string, protocol IngressProtocol) (map[string]map[string]*Ingress, error) {
	namespaceAndIngs := make(map[string]map[string]*Ingress)
	for port, conf := range configs {
		ingressPort, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid ingress config with invalid port:%s", port)
		}

		namespace, ing, err := configToIngress(ingressPort, conf, protocol)
		if err != nil {
			return nil, err
		}
		ings, ok := namespaceAndIngs[namespace]
		if ok == false {
			ings = make(map[string]*Ingress)
			namespaceAndIngs[namespace] = ings
		}
		if old, ok := ings[ing.Name]; ok {
			ing.Rules = append(ing.Rules, old.Rules...)
		}
		ings[ing.Name] = ing
	}
	return namespaceAndIngs, nil
}

func configToIngress(ingressPort int, data string, protocol IngressProtocol) (string, *Ingress, error) {
	serviceAndPort := strings.Split(data, ":")
	if len(serviceAndPort) != 2 {
		return "", nil, fmt.Errorf("invalid tansport layer ingress:%s", data)
	}

	namespaceAndSvc := strings.Split(serviceAndPort[0], "/")
	if len(namespaceAndSvc) != 2 {
		return "", nil, fmt.Errorf("invalid tansport layer ingress:%s", data)
	}

	servicePort, _ := strconv.Atoi(serviceAndPort[1])
	serviceName := namespaceAndSvc[1]
	return namespaceAndSvc[0], &Ingress{
		Name: serviceName,
		Rules: []IngressRule{
			IngressRule{
				Port:     ingressPort,
				Protocol: protocol,
				Paths: []IngressPath{
					IngressPath{
						ServiceName: serviceName,
						ServicePort: servicePort,
					},
				},
			},
		},
	}, nil
}

func ingressLinkedServices(ing *Ingress) StringSet {
	ss := NewStringSet()
	for _, rule := range ing.Rules {
		for _, path := range rule.Paths {
			ss.Add(path.ServiceName)
		}
	}
	return ss
}

func ingressRemoveRules(ing *Ingress, protocol IngressProtocol) {
	var rulesToKeep []IngressRule
	for _, rule := range ing.Rules {
		if rule.Protocol != protocol {
			rulesToKeep = append(rulesToKeep, rule)
		}
	}
	ing.Rules = rulesToKeep
}

func protocolForConfigMap(name string) IngressProtocol {
	switch name {
	case NginxUDPConfigMapName:
		return IngressProtocolUDP
	case NginxTCPConfigMapName:
		return IngressProtocolTCP
	default:
		panic("should pass other configmap here")
	}
}
