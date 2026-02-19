package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Current state info metric
	stateGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "modem_monitor_state",
		Help: "Current state of the modem monitor (0=operating 1=investigating 2=resetting 3=recovering)",
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
// all others to 0. Also updates the state info metric.
func SetState(state string) {
	for i, s := range allStates {
		if s == state {
			stateGauge.Set(float64(i))
		}
	}
}
