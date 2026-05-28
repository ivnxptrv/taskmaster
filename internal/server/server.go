package server

import (
	"fmt"
	"taskmaster/internal/engine"
)

// Gateway of engine will be used by server

type Server struct {
	g *engine.Gateway
}

func NewServer(g *engine.Gateway) *Server {
	return &Server{g: g}
}

// //// 1. Create the engine manager
//     mgr := engine.NewManager()

//     // 2. Create the server and inject the manager
//     // This allows you to easily swap the server for something else later
//     srv := network.NewServer(mgr)

//     // 3. Start them
//     go srv.Listen()
//     mgr.Run()
