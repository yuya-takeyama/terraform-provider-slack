package internal

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/slack-go/slack"
	"go.uber.org/mock/gomock"

	"github.com/sivchari/terraform-provider-slack/internal/appmanifest"
	"github.com/sivchari/terraform-provider-slack/internal/mock"
)

func TestAccAppResource(t *testing.T) {
	t.Parallel()

	createResp := &appmanifest.CreateResponse{
		AppID: "A012345678",
		Credentials: appmanifest.Credentials{
			ClientID:          "1234567890.1234567890123",
			ClientSecret:      "abcdefghijklmnopqrstuvwxyz012345",
			VerificationToken: "abcdefghijklmnopqrstuvwx",
			SigningSecret:     "0123456789abcdef0123456789abcdef",
		},
		OAuthAuthorizeURL: "https://slack.com/oauth/v2/authorize?client_id=1234567890.1234567890123",
	}

	ctrl := gomock.NewController(t)
	client := mock.NewMockAPIClient(ctrl)

	client.EXPECT().CreateAppManifest(gomock.Any(), gomock.Any(), "").Return(createResp, nil).AnyTimes()
	client.EXPECT().DeleteManifestContext(gomock.Any(), "", "A012345678").Return(&slack.SlackResponse{Ok: true}, nil).AnyTimes()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(client),
		Steps: []resource.TestStep{
			{
				Config: testAccAppResource(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("slack_app.test", "id", "A012345678"),
					resource.TestCheckResourceAttr("slack_app.test", "client_id", "1234567890.1234567890123"),
					resource.TestCheckResourceAttr("slack_app.test", "client_secret", "abcdefghijklmnopqrstuvwxyz012345"),
					resource.TestCheckResourceAttr("slack_app.test", "verification_token", "abcdefghijklmnopqrstuvwx"),
					resource.TestCheckResourceAttr("slack_app.test", "signing_secret", "0123456789abcdef0123456789abcdef"),
					resource.TestCheckResourceAttr("slack_app.test", "oauth_authorize_url", "https://slack.com/oauth/v2/authorize?client_id=1234567890.1234567890123"),
					resource.TestCheckResourceAttr("slack_app.test", "display_information.name", "test"),
					resource.TestCheckResourceAttr("slack_app.test", "features.bot_user.display_name", "test-bot"),
					resource.TestCheckResourceAttr("slack_app.test", "oauth_config.scopes.bot.0", "chat:write"),
				),
			},
		},
	})
}

func TestAccAppResourceWithoutBotToken(t *testing.T) {
	t.Parallel()

	createResp := &appmanifest.CreateResponse{
		AppID: "A012345678",
	}

	ctrl := gomock.NewController(t)
	client := mock.NewMockAPIClient(ctrl)

	client.EXPECT().CreateAppManifest(gomock.Any(), gomock.Any(), "").Return(createResp, nil).AnyTimes()
	client.EXPECT().DeleteManifestContext(gomock.Any(), "", "A012345678").Return(&slack.SlackResponse{Ok: true}, nil).AnyTimes()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(client),
		Steps: []resource.TestStep{
			{
				Config: providerConfigNoToken + testAccAppResourceConfig(),
				Check:  resource.TestCheckResourceAttr("slack_app.test", "id", "A012345678"),
			},
		},
	})
}

func testAccAppResource() string {
	return providerConfig + testAccAppResourceConfig()
}

func testAccAppResourceConfig() string {
	return `
resource "slack_app" "test" {
	display_information = {
		name = "test"
	}
	features = {
		bot_user = {
			display_name = "test-bot"
		}
	}
	oauth_config = {
		scopes = {
			bot = ["chat:write"]
		}
	}
}`
}

// TestAppResourceUpdateUsesStateAppID covers an app adopted with
// "terraform import": its computed id is unknown in the plan, so the app_id
// sent to apps.manifest.update has to be read from the prior state.
func TestAppResourceUpdateUsesStateAppID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)
	client := mock.NewMockAPIClient(ctrl)
	// gomock fails the test if app_id is anything but the state's app ID.
	client.EXPECT().
		UpdateManifestContext(gomock.Any(), gomock.Any(), "", "A012345678").
		Return(&slack.UpdateManifestResponse{AppId: "A012345678"}, nil)

	r := &ResourceApp{client: client}
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	prior := ResourceAppState{
		ID:                 types.StringValue("A012345678"),
		DisplayInformation: &AppDisplayInformation{Name: types.StringValue("test")},
	}
	planned := ResourceAppState{
		// as rendered by "(known after apply)" for a computed attribute
		ID:                 types.StringUnknown(),
		ClientID:           types.StringUnknown(),
		ClientSecret:       types.StringUnknown(),
		VerificationToken:  types.StringUnknown(),
		SigningSecret:      types.StringUnknown(),
		OAuthAuthorizeURL:  types.StringUnknown(),
		DisplayInformation: &AppDisplayInformation{Name: types.StringValue("renamed")},
	}

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("failed to build the prior state: %v", diags)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("failed to build the plan: %v", diags)
	}

	res := &fwresource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: state}, res)
	if res.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors: %v", res.Diagnostics)
	}

	var got ResourceAppState
	if diags := res.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("failed to read the updated state: %v", diags)
	}
	if got.ID.ValueString() != "A012345678" {
		t.Errorf("id = %v, want A012345678", got.ID)
	}
	if !got.ClientSecret.IsNull() {
		t.Errorf("client_secret = %v, want null", got.ClientSecret)
	}
	if got.DisplayInformation.Name.ValueString() != "renamed" {
		t.Errorf("display_information.name = %v, want renamed", got.DisplayInformation.Name)
	}
}

// TestAppResourceUpdateSurfacesManifestErrors checks that the validation
// errors Slack returns alongside a bare "invalid_manifest" reach the
// diagnostic.
func TestAppResourceUpdateSurfacesManifestErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	failed := &slack.UpdateManifestResponse{
		ManifestResponse: slack.ManifestResponse{
			Errors: []slack.ManifestValidationError{
				{Pointer: "/settings/interactivity", Message: "requires_request_url"},
			},
			SlackResponse: slack.SlackResponse{Error: "invalid_manifest"},
		},
	}

	ctrl := gomock.NewController(t)
	client := mock.NewMockAPIClient(ctrl)
	client.EXPECT().
		UpdateManifestContext(gomock.Any(), gomock.Any(), "", "A012345678").
		Return(failed, failed.Err())

	r := &ResourceApp{client: client}
	schemaResp := &fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, schemaResp)

	prior := ResourceAppState{
		ID:                 types.StringValue("A012345678"),
		DisplayInformation: &AppDisplayInformation{Name: types.StringValue("test")},
	}
	planned := prior
	planned.ID = types.StringUnknown()

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("failed to build the prior state: %v", diags)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, &planned); diags.HasError() {
		t.Fatalf("failed to build the plan: %v", diags)
	}

	res := &fwresource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, fwresource.UpdateRequest{Plan: plan, State: state}, res)

	errs := res.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Fatalf("Update() reported %d errors, want 1: %v", len(errs), res.Diagnostics)
	}
	want := "invalid_manifest\n/settings/interactivity: requires_request_url"
	if got := errs[0].Detail(); got != want {
		t.Errorf("detail = %q, want %q", got, want)
	}
}
