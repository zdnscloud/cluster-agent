package types

type Resource struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}
