package internal

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func objectValue(null bool) tftypes.Value {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}}
	if null {
		return tftypes.NewValue(typ, nil)
	}
	return tftypes.NewValue(typ, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "A012345678")})
}

func TestCreateOnlyValuePlanModifier(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stateNull   bool
		planNull    bool
		stateValue  types.String
		planValue   types.String
		configValue types.String
		want        types.String
	}{
		"create leaves the value unknown": {
			stateNull:   true,
			stateValue:  types.StringNull(),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			want:        types.StringUnknown(),
		},
		"update keeps a known prior value": {
			stateValue:  types.StringValue("A012345678"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			want:        types.StringValue("A012345678"),
		},
		"update keeps a null prior value from an imported app": {
			stateValue:  types.StringNull(),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			want:        types.StringNull(),
		},
		"a known planned value is left alone": {
			stateValue:  types.StringValue("A012345678"),
			planValue:   types.StringValue("A087654321"),
			configValue: types.StringNull(),
			want:        types.StringValue("A087654321"),
		},
		"destroy leaves the value unknown": {
			planNull:    true,
			stateValue:  types.StringValue("A012345678"),
			planValue:   types.StringUnknown(),
			configValue: types.StringNull(),
			want:        types.StringUnknown(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				State:       tfsdk.State{Raw: objectValue(tt.stateNull)},
				Plan:        tfsdk.Plan{Raw: objectValue(tt.planNull)},
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.configValue,
			}
			res := &planmodifier.StringResponse{PlanValue: tt.planValue}

			createOnlyValue().PlanModifyString(context.Background(), req, res)

			if res.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", res.Diagnostics)
			}
			if !res.PlanValue.Equal(tt.want) {
				t.Errorf("got plan value %v, want %v", res.PlanValue, tt.want)
			}
		})
	}
}
