package machine

import (
	"bytes"
	"testing"

	"github.com/retronet-labs/retronet-8080/cpu"
)

func TestProfilesExpose8080Machines(t *testing.T) {
	for _, name := range []string{"generic", "altair-8800", "imsai-8080", "cpm-dev"} {
		profile, ok := Lookup(name)
		if !ok {
			t.Fatalf("profile %q not found", name)
		}
		if len(profile.MemoryRegions) == 0 || profile.MemoryRegions[0].End != cpu.AddressMask {
			t.Fatalf("profile %q memory = %+v", name, profile.MemoryRegions)
		}
	}
}

func TestCallbackIOSupportsAllPorts(t *testing.T) {
	ioBus := NewCallbackIO()
	if err := ioBus.SetInput(255, 0xAB); err != nil {
		t.Fatal(err)
	}
	ioBus.Output(255, 0xCD)
	if got := ioBus.Input(255); got != 0xAB {
		t.Fatalf("Input(255)=0x%02X", got)
	}
	if got, _ := ioBus.OutputValue(255); got != 0xCD {
		t.Fatalf("OutputValue(255)=0x%02X", got)
	}
}

func TestTerminalDefaultPorts(t *testing.T) {
	ioBus := NewCallbackIO()
	var out bytes.Buffer
	terminal := NewTerminal(&out)
	terminal.QueueInputString("Z")
	if err := terminal.Attach(ioBus); err != nil {
		t.Fatal(err)
	}
	if got := ioBus.Input(TerminalInputPort); got != 'Z' {
		t.Fatalf("terminal input=0x%02X", got)
	}
	ioBus.Output(TerminalOutputPort, 'Q')
	if out.String() != "Q" {
		t.Fatalf("terminal output=%q", out.String())
	}
}

func TestMemoryBusProtectsROM(t *testing.T) {
	bus, err := NewMemoryBus([]MemoryRegion{{Name: "all", Start: 0, End: cpu.AddressMask, Kind: MemoryKindMixed}})
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.LoadROM(0x1000, []byte{0xAA}); err != nil {
		t.Fatal(err)
	}
	bus.Write(0x1000, 0x55)
	if got := bus.Read(0x1000); got != 0xAA {
		t.Fatalf("ROM value=0x%02X", got)
	}
}
