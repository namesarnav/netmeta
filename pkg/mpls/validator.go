package mpls

import (
	"fmt"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type LabelStack struct {
	Labels []Label
	Valid  bool
	Error  string
}

type Label struct {
	Value uint32
	BoS   bool
	TTL   uint8
	TC    uint8
}

type Validator struct {
	corruptionEvents int64
	mu               sync.RWMutex
}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidatePacket(packet gopacket.Packet) (*LabelStack, error) {
	mplsLayer := packet.Layer(layers.LayerTypeMPLS)
	if mplsLayer == nil {
		return nil, fmt.Errorf("no MPLS layer found")
	}

	mpls, ok := mplsLayer.(*layers.MPLS)
	if !ok {
		return nil, fmt.Errorf("invalid MPLS layer type")
	}

	stack := &LabelStack{
		Labels: make([]Label, 0),
		Valid:  true,
	}

	// Parse MPLS label stack
	current := mpls
	for current != nil {
		label := Label{
			Value: current.Label,
			BoS:   current.BottomOfStack,
			TTL:   current.TTL,
			TC:    current.TrafficClass,
		}

		// Validate label value (16-1048575)
		if label.Value < 16 || label.Value > 1048575 {
			stack.Valid = false
			stack.Error = fmt.Sprintf("invalid label value: %d (must be 16-1048575)", label.Value)
			v.recordCorruption()
			return stack, fmt.Errorf("invalid label value: %d", label.Value)
		}

		// Validate TTL
		if label.TTL == 0 {
			stack.Valid = false
			stack.Error = "TTL expired"
			v.recordCorruption()
			return stack, fmt.Errorf("TTL expired")
		}

		stack.Labels = append(stack.Labels, label)

		// Check if this is the bottom of stack
		if label.BoS {
			break
		}

		// Try to get next MPLS layer (if stacked)
		nextLayer := packet.Layer(layers.LayerTypeMPLS)
		if nextLayer == nil {
			break
		}
		current, ok = nextLayer.(*layers.MPLS)
		if !ok {
			break
		}
	}

	return stack, nil
}

func (v *Validator) ValidateLabelStack(labels []uint32) error {
	if len(labels) == 0 {
		return fmt.Errorf("empty label stack")
	}

	for i, label := range labels {
		// Validate label value
		if label < 16 || label > 1048575 {
			v.recordCorruption()
			return fmt.Errorf("invalid label value at position %d: %d (must be 16-1048575)", i, label)
		}
	}

	return nil
}

func (v *Validator) recordCorruption() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.corruptionEvents++
}

func (v *Validator) GetCorruptionCount() int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.corruptionEvents
}

func (v *Validator) ResetCorruptionCount() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.corruptionEvents = 0
}

