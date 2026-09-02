package internal

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/slack-go/slack"
)

// socketModeManifest is what apps.manifest.export returns for a socket mode
// app: interactivity is on, but neither URL is configured, and slack.Manifest
// decodes the absent URLs into empty Go strings.
func socketModeManifest() *slack.Manifest {
	m := &slack.Manifest{}
	m.Display.Name = "test"
	m.Settings.Interactivity = slack.Interactivity{IsEnabled: true}
	m.Settings.EventSubscriptions = slack.EventSubscriptions{BotEvents: []string{"app_mention"}}
	return m
}

func TestStateFromManifestEmptyURLsBecomeNull(t *testing.T) {
	t.Parallel()

	state, diags := stateFromManifest(context.Background(), socketModeManifest(), ResourceAppState{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if state.Settings == nil || state.Settings.Interactivity == nil {
		t.Fatal("settings.interactivity is missing from the state")
	}

	interactivity := state.Settings.Interactivity
	if !interactivity.IsEnabled.Equal(types.BoolValue(true)) {
		t.Errorf("is_enabled: got %v, want true", interactivity.IsEnabled)
	}
	if !interactivity.RequestURL.IsNull() {
		t.Errorf("interactivity.request_url: got %v, want null", interactivity.RequestURL)
	}
	if !interactivity.MessageMenuOptionsURL.IsNull() {
		t.Errorf("interactivity.message_menu_options_url: got %v, want null", interactivity.MessageMenuOptionsURL)
	}

	if state.Settings.EventSubscriptions == nil {
		t.Fatal("settings.event_subscriptions is missing from the state")
	}
	if !state.Settings.EventSubscriptions.RequestURL.IsNull() {
		t.Errorf("event_subscriptions.request_url: got %v, want null", state.Settings.EventSubscriptions.RequestURL)
	}
}

func TestStateFromManifestKeepsConfiguredURLs(t *testing.T) {
	t.Parallel()

	m := &slack.Manifest{}
	m.Display.Name = "test"
	m.Settings.Interactivity = slack.Interactivity{
		IsEnabled:             true,
		RequestUrl:            "https://example.com/slack/interactivity",
		MessageMenuOptionsUrl: "https://example.com/slack/options",
	}
	m.Settings.EventSubscriptions = slack.EventSubscriptions{
		RequestUrl: "https://example.com/slack/events",
		BotEvents:  []string{"app_mention"},
	}

	state, diags := stateFromManifest(context.Background(), m, ResourceAppState{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	got := map[string]types.String{
		"interactivity.request_url":              state.Settings.Interactivity.RequestURL,
		"interactivity.message_menu_options_url": state.Settings.Interactivity.MessageMenuOptionsURL,
		"event_subscriptions.request_url":        state.Settings.EventSubscriptions.RequestURL,
	}
	want := map[string]string{
		"interactivity.request_url":              "https://example.com/slack/interactivity",
		"interactivity.message_menu_options_url": "https://example.com/slack/options",
		"event_subscriptions.request_url":        "https://example.com/slack/events",
	}
	for name, value := range got {
		if !value.Equal(types.StringValue(want[name])) {
			t.Errorf("%s: got %v, want %q", name, value, want[name])
		}
	}
}

// TestManifestFromStateNullURLs covers the other direction: a state that
// carries null URLs must still produce a manifest Slack accepts.
func TestManifestFromStateNullURLs(t *testing.T) {
	t.Parallel()

	state, diags := stateFromManifest(context.Background(), socketModeManifest(), ResourceAppState{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	manifest, diags := manifestFromState(context.Background(), state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := manifest.Settings.Interactivity.RequestUrl; got != "" {
		t.Errorf("interactivity.request_url: got %q, want empty", got)
	}
	if got := manifest.Settings.Interactivity.MessageMenuOptionsUrl; got != "" {
		t.Errorf("interactivity.message_menu_options_url: got %q, want empty", got)
	}
	if !manifest.Settings.Interactivity.IsEnabled {
		t.Error("interactivity.is_enabled: got false, want true")
	}
	if got := manifest.Settings.EventSubscriptions.RequestUrl; got != "" {
		t.Errorf("event_subscriptions.request_url: got %q, want empty", got)
	}
}
