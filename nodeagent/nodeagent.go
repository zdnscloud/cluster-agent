package nodeagent

import (
	"sync"

	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
)

type NodeAgentManager struct {
	lock       sync.Mutex
	nodeAgents map[string]*NodeAgent
}

func New() *NodeAgentManager {
	return &NodeAgentManager{
		nodeAgents: make(map[string]*NodeAgent),
	}
}

func (m *NodeAgentManager) List(ctx *resource.Context) interface{} {
	return m.GetNodeAgents()

}

func (m *NodeAgentManager) GetNodeAgents() []*NodeAgent {
	m.lock.Lock()
	defer m.lock.Unlock()

	var nodes []*NodeAgent
	for _, node := range m.nodeAgents {
		nodes = append(nodes, node)
	}
	return nodes
}

func (m *NodeAgentManager) GetNodeAgent(name string) (*NodeAgent, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	agent, ok := m.nodeAgents[name]
	return agent, ok
}

func (m *NodeAgentManager) Create(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	node := ctx.Resource.(*NodeAgent)
	node.SetID(node.Name)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.nodeAgents[node.Name] = node
	return node, nil
}

func (m *NodeAgentManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, NodeAgent{}, m)
}
