package tools

import (
	"fmt"

	"github.com/x-tymus/x-tymus/core"
)

// RunStealthAITest is a helper callable from tests/tools.
func RunStealthAITest() {
	score, err := core.AnalyzeTrafficWithStealthAI("Test-UA|127.0.0.1|/testpath")
	if err != nil {
		fmt.Printf("AnalyzeTrafficWithStealthAI error: %v\n", err)
		return
	}
	fmt.Printf("StealthAI score: %f\n", score)
}
