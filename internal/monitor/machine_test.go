package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockPinger allows tests to control ping results.
type mockPinger struct {
	mu      sync.Mutex
	results []bool
	index   int
	calls   atomic.Int32
}

func newMockPinger(results ...bool) *mockPinger {
	return &mockPinger{results: results}
}

func (m *mockPinger) PingAll(ctx context.Context) bool {
	m.calls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index >= len(m.results) {
		return m.results[len(m.results)-1]
	}
	result := m.results[m.index]
	m.index++
	return result
}

func (m *mockPinger) setResults(results ...bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results = results
	m.index = 0
}

// mockGPIO records GPIO calls.
type mockGPIO struct {
	mu        sync.Mutex
	highCalls int
	lowCalls  int
	isHigh    bool
}

func (m *mockGPIO) High() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.highCalls++
	m.isHigh = true
	return nil
}

func (m *mockGPIO) Low() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lowCalls++
	m.isHigh = false
	return nil
}

func (m *mockGPIO) Close() error {
	return m.Low()
}

func (m *mockGPIO) getHighCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.highCalls
}

func (m *mockGPIO) getLowCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lowCalls
}

func fastConfig() Config {
	return Config{
		PingInterval:        10 * time.Millisecond,
		InvestigateInterval: 5 * time.Millisecond,
		InvestigateDuration: 30 * time.Millisecond,
		ResetDuration:       20 * time.Millisecond,
		RecoverTimeout:      100 * time.Millisecond,
	}
}

func TestOperatingToInvestigating(t *testing.T) {
	// Ping succeeds twice, then fails → should transition to Investigating
	p := newMockPinger(true, true, false)
	g := &mockGPIO{}
	sm := New(p, g, fastConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Run in a goroutine, check state transitions
	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	// Wait for the state machine to reach Investigating
	deadline := time.After(150 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for Investigating state")
		default:
			if sm.state == Investigating {
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestInvestigatingBackToOperating(t *testing.T) {
	// Ping fails once (go to investigating), then succeeds → back to Operating
	p := newMockPinger(false, false, true)
	g := &mockGPIO{}
	sm := New(p, g, fastConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	// Wait for the state machine to return to Operating after Investigating
	deadline := time.After(150 * time.Millisecond)
	sawInvestigating := false
	for {
		select {
		case <-deadline:
			if !sawInvestigating {
				t.Fatal("never reached Investigating state")
			}
			t.Fatal("timed out waiting for return to Operating state")
		default:
			if sm.state == Investigating {
				sawInvestigating = true
			}
			if sawInvestigating && sm.state == Operating {
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestInvestigatingToResetting(t *testing.T) {
	// All pings fail → should go through Investigating to Resetting
	p := newMockPinger(false)
	g := &mockGPIO{}
	sm := New(p, g, fastConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	deadline := time.After(400*time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for Resetting state")
		default:
			if sm.state == Resetting || g.getHighCalls() > 0 {
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestResettingToRecovering(t *testing.T) {
	// All pings fail → goes through resetting and into recovering
	p := newMockPinger(false)
	g := &mockGPIO{}
	sm := New(p, g, fastConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	deadline := time.After(400 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for Recovering state")
		default:
			if sm.state == Recovering {
				if g.getHighCalls() < 1 {
					t.Fatal("expected GPIO High to be called during reset")
				}
				if g.getLowCalls() < 1 {
					t.Fatal("expected GPIO Low to be called after reset")
				}
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestRecoveringToOperating(t *testing.T) {
	// Pings fail until we're in recovering, then succeed
	callCount := atomic.Int32{}
	p := &funcPinger{fn: func(ctx context.Context) bool {
		n := callCount.Add(1)
		// Fail for the first several pings, then succeed
		return n > 8
	}}
	g := &mockGPIO{}
	sm := New(p, g, fastConfig())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	deadline := time.After(800 * time.Millisecond)
	sawRecovering := false
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out, sawRecovering=%v, state=%s", sawRecovering, sm.state)
		default:
			if sm.state == Recovering {
				sawRecovering = true
			}
			if sawRecovering && sm.state == Operating {
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestRecoverTimeoutResetsAgain(t *testing.T) {
	// Pings always fail → should reset, recover, timeout, reset again
	p := newMockPinger(false)
	g := &mockGPIO{}
	cfg := fastConfig()
	cfg.RecoverTimeout = 30 * time.Millisecond
	sm := New(p, g, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	deadline := time.After(800 * time.Millisecond)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for second reset, highCalls=%d", g.getHighCalls())
		default:
			if g.getHighCalls() >= 2 {
				cancel()
				<-done
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}
}

func TestShutdownRestoresPower(t *testing.T) {
	// Ensure GPIO is set low when context is cancelled during reset
	p := newMockPinger(false)
	g := &mockGPIO{}
	cfg := fastConfig()
	cfg.ResetDuration = time.Second // Long reset so we can cancel during it
	sm := New(p, g, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sm.Run(ctx)
		close(done)
	}()

	<-done

	// After shutdown, the relay should be low (power restored)
	if g.getLowCalls() < 1 {
		t.Fatal("expected GPIO Low to be called on shutdown")
	}
}

// funcPinger is a Pinger backed by a function, for flexible test control.
type funcPinger struct {
	fn func(ctx context.Context) bool
}

func (f *funcPinger) PingAll(ctx context.Context) bool {
	return f.fn(ctx)
}
