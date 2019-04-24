package storage

const (
	LogLevel = "debug"
	CtrlName = "StorageCtrl"

	ZkeStorageLabel     = "node-role.kubernetes.io/storage"
	ZkeInternalIPAnnKey = "zdnscloud.cn/internal-ip"
	LvmdPort            = "1736"
	ConTimeout          = 10
	DefaultVgName       = "k8s"
	LVM                 = "lvm"

	ZKEStorageNamespace = "zcloud"
	ZKENFSPvcName       = "nfs-data-nfs-provisioner-0"
	NFS                 = "nfs"
)
