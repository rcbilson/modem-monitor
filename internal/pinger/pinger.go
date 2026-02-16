package pinger

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Pinger tests connectivity to a set of targets.
// Returns true if any target responds successfully.
type Pinger interface {
	PingAll(ctx context.Context) bool
}

// ICMPPinger sends ICMP pings using pro-bing.
type ICMPPinger struct {
	Targets []string
	Timeout time.Duration
}

// NewICMPPinger creates a pinger for the given targets.
func NewICMPPinger(targets []string) *ICMPPinger {
	return &ICMPPinger{
		Targets: targets,
		Timeout: 3 * time.Second,
	}
}

// PingAll pings all targets concurrently. Returns true if any respond.
func (p *ICMPPinger) PingAll(ctx context.Context) bool {
	var (
		wg      sync.WaitGroup
		success atomic.Bool
	)

	for _, target := range p.Targets {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			pinger, err := probing.NewPinger(addr)
			if err != nil {
				return
			}
			pinger.Count = 1
			pinger.Timeout = p.Timeout
			pinger.SetPrivileged(true)

			err = pinger.RunWithContext(ctx)
			if err != nil {
				return
			}

			if pinger.Statistics().PacketsRecv > 0 {
				success.Store(true)
			}
		}(target)
	}

	wg.Wait()
	return success.Load()
}
