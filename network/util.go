package network

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/uuid"
)

func GenUUID() string {
	id, err := uuid.Gen()
	if err != nil {
		log.Fatalf("generate uuid failed:%s", err.Error())
	}
	return id
}
