package gpio

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
)

// Controller controls a relay via GPIO.
type Controller interface {
	High() error // Cut power
	Low() error  // Restore power
	Close() error
}

// RelayController drives a relay using the pinctrl command (Pi 5 / RP1 compatible).
type RelayController struct {
	pin int
}

// NewRelayController opens the given pin and sets it low.
func NewRelayController(pinNumber int) (*RelayController, error) {
	c := &RelayController{pin: pinNumber}

	// Start with relay off (power on to modem)
	if err := c.Low(); err != nil {
		return nil, fmt.Errorf("gpio set low: %w", err)
	}

	log.Printf("GPIO: initialized pin %d", pinNumber)
	return c, nil
}

func (r *RelayController) High() error {
	return pinctrl(r.pin, "dh")
}

func (r *RelayController) Low() error {
	return pinctrl(r.pin, "dl")
}

func (r *RelayController) Close() error {
	return r.Low()
}

func pinctrl(pin int, state string) error {
	out, err := exec.Command("pinctrl", "set", strconv.Itoa(pin), state).CombinedOutput()
	if err != nil {
		return fmt.Errorf("pinctrl set %d %s: %w: %s", pin, state, err, out)
	}
	return nil
}
