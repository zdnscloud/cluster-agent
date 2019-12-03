package types

type Edge struct {
	Src      Resource `json:"src"`
	Dst      Resource `json:"dst"`
	ClientID string   `json:"clientID,omitempty"`
	ServerID string   `json:"serverID,omitempty"`
	Msg      string   `json:"noTLSReason,omitempty"`
}

type Edges []*Edge

func (e Edges) Len() int {
	return len(e)
}

func (e Edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e Edges) Less(i, j int) bool {
	return e[i].Src.Namespace+e[i].Dst.Namespace+e[i].Src.Name+e[i].Dst.Name <
		e[j].Src.Namespace+e[j].Dst.Namespace+e[j].Src.Name+e[j].Dst.Name
}
