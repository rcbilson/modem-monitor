package monitor

import (
	"context"
	"log"
	"time"

	"modem-monitor/internal/gpio"
	"modem-monitor/internal/metrics"
	"modem-monitor/internal/pinger"
)

// Config holds timing parameters for the state machine.
type Config struct {
	PingInterval        time.Duration
	InvestigateInterval time.Duration
	InvestigateDuration time.Duration
	ResetDuration       time.Duration
	RecoverTimeout      time.Duration
}

// StateMachine implements the modem monitor state machine.
type StateMachine struct {
	pinger pinger.Pinger
	gpio   gpio.Controller
	config Config
	state  State
}

// New creates a new StateMachine.
func New(p pinger.Pinger, g gpio.Controller, cfg Config) *StateMachine {
	return &StateMachine{
		pinger: p,
		gpio:   g,
		config: cfg,
		state:  Operating,
	}
}

// Run starts the state machine loop. It blocks until the context is cancelled.
func (sm *StateMachine) Run(ctx context.Context) {
	sm.transition(Operating)

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down state machine")
			return
		default:
		}

		switch sm.state {
		case Operating:
			sm.runOperating(ctx)
		case Investigating:
			sm.runInvestigating(ctx)
		case Resetting:
			sm.runResetting(ctx)
		case Recovering:
			sm.runRecovering(ctx)
		}
	}
}

func (sm *StateMachine) transition(newState State) {
	if sm.state != newState {
		log.Printf("state: %s → %s", sm.state, newState)
		metrics.StateTransitions.Inc()
	}
	sm.state = newState
	metrics.SetState(newState.String())
}

func (sm *StateMachine) ping(ctx context.Context) bool {
	ok := sm.pinger.PingAll(ctx)
	if ok {
		metrics.PingSuccess.Inc()
	} else {
		metrics.PingFailure.Inc()
	}
	return ok
}

// runOperating pings at the normal interval. Transitions to Investigating
// when all pings fail.
func (sm *StateMachine) runOperating(ctx context.Context) {
	ticker := time.NewTicker(sm.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !sm.ping(ctx) {
				sm.transition(Investigating)
				return
			}
		}
	}
}

// runInvestigating pings rapidly to confirm the outage. Any success returns
// to Operating. All failures for InvestigateDuration transitions to Resetting.
func (sm *StateMachine) runInvestigating(ctx context.Context) {
	ticker := time.NewTicker(sm.config.InvestigateInterval)
	defer ticker.Stop()

	deadline := time.After(sm.config.InvestigateDuration)

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			sm.transition(Resetting)
			return
		case <-ticker.C:
			if sm.ping(ctx) {
				sm.transition(Operating)
				return
			}
		}
	}
}

// runResetting cuts modem power via GPIO, waits, then restores power.
func (sm *StateMachine) runResetting(ctx context.Context) {
	metrics.Resets.Inc()

	log.Println("cutting modem power")
	if err := sm.gpio.High(); err != nil {
		log.Printf("gpio high error: %v", err)
	}

	select {
	case <-ctx.Done():
		// Restore power on shutdown
		sm.gpio.Low()
		return
	case <-time.After(sm.config.ResetDuration):
	}

	log.Println("restoring modem power")
	if err := sm.gpio.Low(); err != nil {
		log.Printf("gpio low error: %v", err)
	}

	sm.transition(Recovering)
}

// runRecovering waits for connectivity to return. If it doesn't come back
// within RecoverTimeout, transitions back to Resetting.
func (sm *StateMachine) runRecovering(ctx context.Context) {
	ticker := time.NewTicker(sm.config.PingInterval)
	defer ticker.Stop()

	deadline := time.After(sm.config.RecoverTimeout)

	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			log.Println("recovery timeout, resetting again")
			sm.transition(Resetting)
			return
		case <-ticker.C:
			if sm.ping(ctx) {
				sm.transition(Operating)
				return
			}
		}
	}
}
