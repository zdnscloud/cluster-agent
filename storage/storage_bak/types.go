package storage

import (
	resttypes "github.com/zdnscloud/gorest/types"
	//runtime "k8s.io/apimachinery/pkg/runtime"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var StorageType = resttypes.GetResourceType(Storage{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	TotalSize          int    `json:"totalsize,omitempty"`
	FreeSize           int    `json:"freesize,omitempty"`
	PVs                []Pv   `json:"pvs,omitempty"`
	Nodes              []Node `json:"nodes,omitempty"`
}

type Node struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	Addr               string `json:"addr,omitempty"`
	TotalSize          int    `json:"totalsize,omitempty"`
	FreeSize           int    `json:"freesize,omitempty"`
	Vgs                []VG   `json:"vgs,omitempty"`
}

type VG struct {
	resttypes.Resource `json:",inline"`
	Name               string   `json:"name,omitempty"`
	Size               int      `json:"size,omitempty"`
	FreeSize           int      `json:"free_size,omitempty"`
	Uuid               string   `json:"uuid,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

/*
type Pod struct {
	Name string `json:"name,omitempty"`
}*/

type Pv struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	Size               int    `json:"size,omitempty"`
	//Pods               []corev1.Pod.ObjectMeta.Name  `json:"pods,omitempty"`
	Pods []string `json:"pods,omitempty"`
	Pvc  string   `json:"pvc,omitempty"`
}

/*
func (in *Storage) DeepCopyInto(out *Storage) {
	*out = *in
	if in.PVs != nil {
		in, out := &in.PVs, &out.PVs
		*out = make([]Pv, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make([]Node, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

func (in *Storage) DeepCopy() *Storage {
	if in == nil {
		return nil
	}
	out := new(Storage)
	in.DeepCopyInto(out)
	return out
}
func (in *Pv) DeepCopyInto(out *Pv) {
	*out = *in
	if in.Pods != nil {
		in, out := &in.Pods, &out.Pods
		*out = make([]Pod, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

func (in *Pv) DeepCopy() *Pv {
	if in == nil {
		return nil
	}
	out := new(Pv)
	in.DeepCopyInto(out)
	return out
}
func (in *Node) DeepCopyInto(out *Node) {
	*out = *in
	if in.Vgs != nil {
		in, out := &in.Vgs, &out.Vgs
		*out = make([]VG, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

func (in *Node) DeepCopy() *Node {
	if in == nil {
		return nil
	}
	out := new(Node)
	in.DeepCopyInto(out)
	return out
}

func (in *VG) DeepCopyInto(out *VG) {
	*out = *in
	return
}

func (in *VG) DeepCopy() *VG {
	if in == nil {
		return nil
	}
	out := new(VG)
	in.DeepCopyInto(out)
	return out
}
func (in *Pod) DeepCopyInto(out *Pod) {
	*out = *in
	return
}

func (in *Pod) DeepCopy() *Pod {
	if in == nil {
		return nil
	}
	out := new(Pod)
	in.DeepCopyInto(out)
	return out
}

func (in *Storage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
*/
