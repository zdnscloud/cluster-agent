package handler

import (
	"context"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

func createClient() (client.Client, error) {
	config, err := config.GetConfig()
	if err != nil {
		Fatal("get k8s config failed:%s", err.Error())
		return nil, err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		Fatal("create k8s client failed:%s", err.Error())
		return nil, err
	}
	return cli, nil
}

func GetStorageClass() ([]string, error) {
	cli, err := createClient()
	if err != nil {
		return nil, err
	}
	storages := storagev1.StorageClassList{}
	cli.List(context.TODO(), nil, &storages)
	var res []string
	for _, s := range storages.Items {
		res = append(res, s.Name)
	}
	return res, nil

}

func GetNode() ([]corev1.Node, error) {
	cli, err := createClient()
	if err != nil {
		return nil, err
	}
	nodes := corev1.NodeList{}
	cli.List(context.TODO(), nil, &nodes)
	var res []corev1.Node
	for _, n := range nodes.Items {
		v, ok := n.Labels[ZkeStorageLabel]
		if ok && v == "true" {
			res = append(res, n)
		}
	}
	return res, nil
}
