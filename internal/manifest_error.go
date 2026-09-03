package internal

import (
	"strings"

	"github.com/slack-go/slack"
)

// manifestErrorDetail renders the detail of a diagnostic for a failed
// apps.manifest.* call. Slack reports the top level reason as a bare error
// code such as "invalid_manifest" and puts the actionable part in the
// response "errors" array, so the code on its own is rarely enough to tell
// what is wrong with the manifest.
func manifestErrorDetail(err error, errs []slack.ManifestValidationError) string {
	lines := make([]string, 0, len(errs)+1)
	if err != nil {
		lines = append(lines, err.Error())
	}
	for _, e := range errs {
		if e.Pointer == "" {
			lines = append(lines, e.Message)
			continue
		}
		lines = append(lines, e.Pointer+": "+e.Message)
	}
	return strings.Join(lines, "\n")
}
