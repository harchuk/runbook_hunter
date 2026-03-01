package brief

import (
	"fmt"
	"strings"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/executor"
)

// Security notes: brief is generated from observed facts only; no secrets are embedded.
func Build(alertName, service, env string, results []executor.Result) string {
	ok := 0
	errCount := 0
	facts := make([]string, 0, len(results))
	for _, r := range results {
		if r.Status == "ok" {
			ok++
			facts = append(facts, fmt.Sprintf("%s: ok", r.StepName))
		} else {
			errCount++
			facts = append(facts, fmt.Sprintf("%s: %s", r.StepName, r.Error))
		}
	}
	if len(facts) == 0 {
		return fmt.Sprintf("No runbook steps executed for %s (%s/%s)", alertName, service, env)
	}
	return fmt.Sprintf("%s (%s/%s): %d ok, %d errors. %s", alertName, service, env, ok, errCount, strings.Join(facts, "; "))
}
