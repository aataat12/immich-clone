package main

import (
	"context"
	"golang.org/x/sys/windows/svc"
)

type service struct{ o options }

func (s *service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	close, e := start(ctx, s.o)
	if e != nil {
		return false, 1
	}
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for request := range requests {
		switch request.Cmd {
		case svc.Interrogate:
			status <- request.CurrentStatus
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.StopPending}
			cancel()
			if close() != nil {
				return false, 2
			}
			return false, 0
		}
	}
	cancel()
	close()
	return false, 0
}
func runService(o options) error { return svc.Run("ImmichClone", &service{o}) }
