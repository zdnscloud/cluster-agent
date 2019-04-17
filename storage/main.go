package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cluster-agent/storage/handler"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	app := handler.NewApp()
	router := gin.New()

	if err := app.RegisterHandler(router); err != nil {
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
	server, err := NewServer()
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	addr := "202.173.9.10:3456"
	if err := server.Run(addr); err != nil {
		panic("server run failed:" + err.Error())
	}
}
