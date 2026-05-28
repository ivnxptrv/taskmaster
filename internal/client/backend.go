package client

import (
// "fmt"
// "taskmaster/internal/engine"
)

// for direct usage Gateway of engine used as Backend directly

// -----------------------------------------------

// -----------------------------------------------

type Network struct{}

func NewNetwork(url string) *Network {
	return &Network{}
}

func (n *Network) Dial() {}

func (n *Network) Start(name string) {}
