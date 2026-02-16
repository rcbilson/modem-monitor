package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Labeled gauge for compatibility with existing queries
	stateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "modem_monitor_state",
		Help: "Current state of the modem monitor (1=active, 0=inactive)",
	}, []string{"state"})

	// Single numeric gauge for timeline visualization
	// 0=operating, 1=investigating, 2=resetting, 3=recovering
	stateNumeric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "modem_monitor_state_numeric",
		Help: "Current state as numeric value: 0=operating, 1=investigating, 2=resetting, 3=recovering",
	})

	StateTransitions = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modem_monitor_state_transitions_total",
		Help: "Total number of state transitions",
	})

	Resets = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modem_monitor_resets_total",
		Help: "Total number of modem resets",
	})

	PingSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modem_monitor_ping_success_total",
		Help: "Total number of successful ping rounds",
	})

	PingFailure = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modem_monitor_ping_failure_total",
		Help: "Total number of failed ping rounds",
	})

	allStates = []string{"operating", "investigating", "resetting", "recovering"}
)

// SetState sets the active state gauge. The given state is set to 1,
// all others to 0. Also sets the numeric state gauge.
func SetState(state string) {
	for _, s := range allStates {
		if s == state {
			stateGauge.WithLabelValues(s).Set(1)
		} else {
			stateGauge.WithLabelValues(s).Set(0)
		}
	}

	// Set numeric value for timeline visualization
	switch state {
	case "operating":
		stateNumeric.Set(0)
	case "investigating":
		stateNumeric.Set(1)
	case "resetting":
		stateNumeric.Set(2)
	case "recovering":
		stateNumeric.Set(3)
	}
}
