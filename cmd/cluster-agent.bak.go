package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/storage"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	//app := storage.NewApp()
	if err := storage.RegisterHandler(router); err != nil {
		panic("register handler failed:" + err.Error())
	}
	if err := network.RegisterHandler(router); err != nil {
		panic("register handler failed:" + err.Error())
	}

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	//app := storage.NewApp()
	if err := storage.RegisterHandler(router); err != nil {
		panic("register handler failed:" + err.Error())
	}
	if err := network.RegisterHandler(router); err != nil {
		panic("register handler failed:" + err.Error())
	}

	return &Server{
		router: router,
	}, nil
	server, err := NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	addr := "0.0.0.0:8090"
	if err := server.Run(addr); err != nil {
		panic("server run failed:" + err.Error())
	}
}
