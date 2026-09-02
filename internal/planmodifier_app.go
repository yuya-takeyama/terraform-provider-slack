package internal

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

var _ planmodifier.String = createOnlyValueModifier{}

// createOnlyValue keeps the prior state of a computed attribute whose value is
// decided once, when the app is created, and is never returned by the Slack
// API again: the app ID and the credentials from apps.manifest.create.
//
// The framework marks every null computed attribute as unknown while planning,
// so an app adopted with "terraform import" - which cannot recover the
// credentials - renders them as "(known after apply)" on every single plan.
// stringplanmodifier.UseStateForUnknown does not help there because it bails
// out when the prior state value is null, which is precisely the imported
// case. This modifier keeps the prior value whatever it is, including null,
// and only steps aside while the resource is being created.
func createOnlyValue() planmodifier.String {
	return createOnlyValueModifier{}
}

type createOnlyValueModifier struct{}

func (createOnlyValueModifier) Description(_ context.Context) string {
	return "Once set, the value of this attribute in state will not change."
}

func (createOnlyValueModifier) MarkdownDescription(ctx context.Context) string {
	return createOnlyValueModifier{}.Description(ctx)
}

func (createOnlyValueModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
	// The resource is being created: the value genuinely is unknown.
	if req.State.Raw.IsNull() {
		return
	}

	// The resource is being destroyed.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Nothing to carry over when the plan already holds a known value.
	if !req.PlanValue.IsUnknown() {
		return
	}

	// A configuration value that is itself unknown must stay unknown.
	if req.ConfigValue.IsUnknown() {
		return
	}

	res.PlanValue = req.StateValue
}
