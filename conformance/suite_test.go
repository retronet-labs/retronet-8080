package conformance

import "testing"

func TestSyntheticSuitePasses(t *testing.T) {
	result := RunSuite(SyntheticSuite())
	if result.Failed != 0 {
		t.Fatalf("suite failed: %+v", result)
	}
	if result.Passed != len(SyntheticSuite()) {
		t.Fatalf("passed=%d want=%d", result.Passed, len(SyntheticSuite()))
	}
}
