// Package conformance esegue programmi sintetici contro il core 8080.
package conformance

import (
	"fmt"

	"github.com/retronet-labs/retronet-8080/cpu"
	"github.com/retronet-labs/retronet-8080/machine"
)

// Context raccoglie i componenti isolati di un singolo caso.
type Context struct {
	CPU      *cpu.CPU8080
	Memory   *machine.ObservableMemory
	IO       *machine.CallbackIO
	Panel    *machine.FrontPanel
	Debugger *machine.Debugger
}

// Case e' un programma sintetico con setup e verifica dichiarati in Go.
type Case struct {
	Name        string
	Program     []byte
	LoadAddress uint16
	StartPC     uint16
	StepLimit   uint64
	Setup       func(*Context) error
	Verify      func(*Context, machine.DebugRunResult) error
}

// CaseResult conserva un esito senza interrompere il resto della suite.
type CaseResult struct {
	Name       string                  `json:"name"`
	Passed     bool                    `json:"passed"`
	Steps      uint64                  `json:"steps"`
	StopReason machine.DebugStopReason `json:"stop_reason"`
	Error      string                  `json:"error,omitempty"`
}

// SuiteResult riassume tutti i casi eseguiti.
type SuiteResult struct {
	Passed int          `json:"passed"`
	Failed int          `json:"failed"`
	Cases  []CaseResult `json:"cases"`
}

// RunCase costruisce una macchina generica nuova e isolata.
func RunCase(test Case) CaseResult {
	result := CaseResult{Name: test.Name}
	if test.Name == "" {
		result.Error = "nome caso obbligatorio"
		return result
	}
	if err := machine.ValidateRange(test.LoadAddress, len(test.Program)); err != nil {
		result.Error = err.Error()
		return result
	}

	base := cpu.NewFlatMemory()
	memory, err := machine.NewObservableMemory(base)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if err := memory.LoadBytes(test.LoadAddress, test.Program); err != nil {
		result.Error = err.Error()
		return result
	}
	c := cpu.NewCPU8080()
	c.PC = test.StartPC
	ioBus := machine.NewCallbackIO()
	panel, err := machine.NewFrontPanel(c, memory, ioBus)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	debugger, err := machine.NewDebugger(panel, memory, ioBus)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	context := &Context{CPU: c, Memory: memory, IO: ioBus, Panel: panel, Debugger: debugger}
	if test.Setup != nil {
		if err := test.Setup(context); err != nil {
			result.Error = fmt.Sprintf("setup: %v", err)
			return result
		}
	}
	limit := test.StepLimit
	if limit == 0 {
		limit = 64
	}
	runResult, err := debugger.Run(limit)
	result.Steps = runResult.Steps
	result.StopReason = runResult.Reason
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if test.Verify != nil {
		if err := test.Verify(context, runResult); err != nil {
			result.Error = err.Error()
			return result
		}
	}
	result.Passed = true
	return result
}

// RunSuite esegue tutti i casi e conserva anche i fallimenti.
func RunSuite(cases []Case) SuiteResult {
	result := SuiteResult{Cases: make([]CaseResult, 0, len(cases))}
	for _, test := range cases {
		caseResult := RunCase(test)
		result.Cases = append(result.Cases, caseResult)
		if caseResult.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
	}
	return result
}
