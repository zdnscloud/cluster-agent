package blockdevice

import (
	"github.com/zdnscloud/gorest/resource"
)

type BlockDevice struct {
	resource.ResourceBase `json:",inline"`
	NodeName              string `json:"nodeName"`
	BlockDevices          []Dev  `json:"blockDevices"`
}

type Dev struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Parted     bool   `json:"parted"`
	Filesystem bool   `json:"filesystem"`
	Mount      bool   `json:"mount"`
}

type BlockDevices []BlockDevice

func (s BlockDevices) Len() int           { return len(s) }
func (s BlockDevices) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s BlockDevices) Less(i, j int) bool { return s[i].NodeName < s[j].NodeName }

type Devs []Dev

func (s Devs) Len() int           { return len(s) }
func (s Devs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Devs) Less(i, j int) bool { return s[i].Name < s[j].Name }
