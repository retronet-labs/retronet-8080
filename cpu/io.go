package cpu

// IO e' il bus I/O separato dalla memoria. L'8080 indirizza 256 porte.
type IO interface {
	Input(port byte) byte
	Output(port byte, value byte)
}

// Ports e' una implementazione semplice con latch per tutte le porte.
type Ports struct {
	InputPorts  [256]byte
	OutputPorts [256]byte
}

func NewPorts() *Ports { return &Ports{} }

func IsInputPort(_ byte) bool { return true }

func IsOutputPort(_ byte) bool { return true }

func ValidateInputPort(_ byte) error { return nil }

func ValidateOutputPort(_ byte) error { return nil }

func (p *Ports) SetInput(port byte, value byte) error {
	p.InputPorts[port] = value
	return nil
}

func (p *Ports) Input(port byte) byte { return p.InputPorts[port] }

func (p *Ports) Output(port byte, value byte) { p.OutputPorts[port] = value }

func (p *Ports) OutputValue(port byte) (byte, error) { return p.OutputPorts[port], nil }
