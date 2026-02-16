package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	gpiopkg "modem-monitor/internal/gpio"
	"modem-monitor/internal/monitor"
	"modem-monitor/internal/pinger"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("modem-monitor: ")

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	log.Printf("ping targets: %v", cfg.PingTargets)
	log.Printf("ping interval: %s, investigate: %s/%s, reset: %s, recover timeout: %s",
		cfg.PingInterval, cfg.InvestigateInterval, cfg.InvestigateDuration,
		cfg.ResetDuration, cfg.RecoverTimeout)
	log.Printf("GPIO pin: %d, metrics addr: %s", cfg.GPIOPin, cfg.MetricsAddr)

	// Initialize pinger
	p := pinger.NewICMPPinger(cfg.PingTargets)

	// Initialize GPIO
	g, err := gpiopkg.NewRelayController(cfg.GPIOPin)
	if err != nil {
		log.Fatalf("gpio init error: %v", err)
	}
	defer g.Close()

	// Start Prometheus metrics server
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Printf("metrics server listening on %s", cfg.MetricsAddr)
		if err := http.ListenAndServe(cfg.MetricsAddr, nil); err != nil {
			log.Fatalf("metrics server error: %v", err)
		}
	}()

	// Set up signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run state machine
	sm := monitor.New(p, g, monitor.Config{
		PingInterval:        cfg.PingInterval,
		InvestigateInterval: cfg.InvestigateInterval,
		InvestigateDuration: cfg.InvestigateDuration,
		ResetDuration:       cfg.ResetDuration,
		RecoverTimeout:      cfg.RecoverTimeout,
	})

	log.Println("starting modem monitor")
	sm.Run(ctx)
	log.Println("modem monitor stopped")
}
