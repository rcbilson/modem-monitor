package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PingTargets         []string
	PingInterval        time.Duration
	InvestigateInterval time.Duration
	InvestigateDuration time.Duration
	ResetDuration       time.Duration
	RecoverTimeout      time.Duration
	GPIOPin             int
	MetricsAddr         string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		PingTargets:         []string{"8.8.8.8", "1.1.1.1"},
		PingInterval:        10 * time.Second,
		InvestigateInterval: 1 * time.Second,
		InvestigateDuration: 10 * time.Second,
		ResetDuration:       10 * time.Second,
		RecoverTimeout:      10 * time.Minute,
		GPIOPin:             4,
		MetricsAddr:         ":9090",
	}

	if v := os.Getenv("PING_TARGETS"); v != "" {
		targets := strings.Split(v, ",")
		cfg.PingTargets = make([]string, 0, len(targets))
		for _, t := range targets {
			t = strings.TrimSpace(t)
			if t != "" {
				cfg.PingTargets = append(cfg.PingTargets, t)
			}
		}
		if len(cfg.PingTargets) == 0 {
			return cfg, fmt.Errorf("PING_TARGETS is set but contains no valid targets")
		}
	}

	var err error

	if cfg.PingInterval, err = envDuration("PING_INTERVAL", cfg.PingInterval); err != nil {
		return cfg, err
	}
	if cfg.InvestigateInterval, err = envDuration("INVESTIGATE_INTERVAL", cfg.InvestigateInterval); err != nil {
		return cfg, err
	}
	if cfg.InvestigateDuration, err = envDuration("INVESTIGATE_DURATION", cfg.InvestigateDuration); err != nil {
		return cfg, err
	}
	if cfg.ResetDuration, err = envDuration("RESET_DURATION", cfg.ResetDuration); err != nil {
		return cfg, err
	}
	if cfg.RecoverTimeout, err = envDuration("RECOVER_TIMEOUT", cfg.RecoverTimeout); err != nil {
		return cfg, err
	}

	if v := os.Getenv("GPIO_PIN"); v != "" {
		cfg.GPIOPin, err = strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid GPIO_PIN: %w", err)
		}
	}

	if v := os.Getenv("METRICS_ADDR"); v != "" {
		cfg.MetricsAddr = v
	}

	return cfg, nil
}

func envDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}
