package engine

import (
	"taskmaster/internal/gateway"
)

// -----------------------------------------------

type Foreman struct {
	m      *Manager
	events chan event
}

func NewForeman(mgr *Manager) *Foreman {
	return &Foreman{m: mgr, events: mgr.Events}
}

func (d *Foreman) Status(name string) gateway.Status {
	// chanResponse := make(chan string)
	// d.events <- Status{Name: name, chanResponse: chanResponse}
	// return <-chanResponse

	// get real info
	// or send event with chanel back to get result
	return gateway.Status{}
}

func (d *Foreman) Start(name string) error {
	d.events <- Start{Name: name}
	return nil
}

func (d *Foreman) Stop(name string) error {
	d.events <- Stop{Name: name}

	return nil
}

func (d *Foreman) Restart(name string) error {
	d.events <- Stop{Name: name}

	return nil
}

func (d *Foreman) Reload() error {
	// d.mgr.Events <- Reload{}
	return nil
}

func (d *Foreman) Shutdown() error {
	res := make(chan struct{})
	d.events <- Shutdown{m: d.m, res: res}
	<-res
	return nil
}

// -----------------------------------------------
