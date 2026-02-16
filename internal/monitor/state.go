package monitor

// State represents the current state of the modem monitor.
type State int

const (
	Operating State = iota
	Investigating
	Resetting
	Recovering
)

func (s State) String() string {
	switch s {
	case Operating:
		return "operating"
	case Investigating:
		return "investigating"
	case Resetting:
		return "resetting"
	case Recovering:
		return "recovering"
	default:
		return "unknown"
	}
}
