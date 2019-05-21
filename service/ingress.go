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

//from business logic
//when update ingress rule
//either http rule will be updated, or none-http rule
//they cann't be mixed in one update

type Ingress struct {
	name  string
	rules []IngressRule
}

type IngressRule struct {
	host     string
	port     int
	protocol IngressProtocol
	paths    []IngressPath
}

type IngressPath struct {
	path        string
	serviceName string
	servicePort int
}

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
		if old, ok := ings[ing.name]; ok {
			ing.rules = append(ing.rules, old.rules...)
		}
		ings[ing.name] = ing
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
		name: serviceName,
		rules: []IngressRule{
			IngressRule{
				port:     ingressPort,
				protocol: protocol,
				paths: []IngressPath{
					IngressPath{
						serviceName: serviceName,
						servicePort: servicePort,
					},
				},
			},
		},
	}, nil
}

func ingressLinkedServices(ing *Ingress) StringSet {
	ss := NewStringSet()
	for _, rule := range ing.rules {
		for _, path := range rule.paths {
			ss.Add(path.serviceName)
		}
	}
	return ss
}

func ingressRemoveRules(ing *Ingress, protocol IngressProtocol) {
	var rulesToKeep []IngressRule
	for _, rule := range ing.rules {
		if protocol == IngressProtocolHTTP {
			if rule.protocol != protocol {
				rulesToKeep = append(rulesToKeep, rule)
			}
		} else {
			if rule.protocol == IngressProtocolHTTP {
				rulesToKeep = append(rulesToKeep, rule)
			}
		}
	}
	ing.rules = rulesToKeep
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
