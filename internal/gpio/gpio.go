package gpio

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// Controller controls a relay via GPIO.
type Controller interface {
	High() error // Cut power
	Low() error  // Restore power
	Close() error
}

// RelayController drives a relay using periph.io GPIO.
type RelayController struct {
	pin gpio.PinOut
}

// NewRelayController initializes the GPIO host and opens the given pin.
func NewRelayController(pinNumber int) (*RelayController, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("gpio host init: %w", err)
	}

	pinName := fmt.Sprintf("GPIO%d", pinNumber)
	pin := gpioreg.ByName(pinName)
	if pin == nil {
		return nil, fmt.Errorf("gpio pin %s not found", pinName)
	}

	// Start with relay off (power on to modem)
	if err := pin.Out(gpio.Low); err != nil {
		return nil, fmt.Errorf("gpio set low: %w", err)
	}

	log.Printf("GPIO: initialized pin %s", pinName)
	return &RelayController{pin: pin}, nil
}

func (r *RelayController) High() error {
	return r.pin.Out(gpio.High)
}

func (r *RelayController) Low() error {
	return r.pin.Out(gpio.Low)
}

func (r *RelayController) Close() error {
	return r.pin.Out(gpio.Low)
}
