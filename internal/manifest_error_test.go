package internal

import (
	"errors"
	"testing"

	"github.com/slack-go/slack"
)

func TestManifestErrorDetail(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		errs []slack.ManifestValidationError
		want string
	}{
		"error only": {
			err:  errors.New("invalid_manifest"),
			want: "invalid_manifest",
		},
		"error with validation errors": {
			err: errors.New("invalid_manifest"),
			errs: []slack.ManifestValidationError{
				{Pointer: "/settings/interactivity", Message: "requires_request_url"},
				{Pointer: "/settings", Message: "requires_socket_mode_disabled"},
			},
			want: "invalid_manifest\n" +
				"/settings/interactivity: requires_request_url\n" +
				"/settings: requires_socket_mode_disabled",
		},
		"validation error without pointer": {
			err:  errors.New("invalid_manifest"),
			errs: []slack.ManifestValidationError{{Message: "something is wrong"}},
			want: "invalid_manifest\nsomething is wrong",
		},
		"no error": {
			errs: []slack.ManifestValidationError{{Pointer: "/settings", Message: "boom"}},
			want: "/settings: boom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := manifestErrorDetail(tt.err, tt.errs); got != tt.want {
				t.Errorf("manifestErrorDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}
