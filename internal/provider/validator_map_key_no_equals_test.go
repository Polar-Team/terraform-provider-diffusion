package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMapKeyNoEquals_ValidKeys(t *testing.T) {
	ctx := context.Background()

	mapVal, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"default": "key1",
		"backup":  "key2",
		"my-host": "key3",
	})
	if diags.HasError() {
		t.Fatalf("unexpected error creating map: %s", diags.Errors())
	}

	req := validator.MapRequest{
		Path:        path.Root("ssh_private_keys"),
		ConfigValue: mapVal,
	}
	resp := &validator.MapResponse{}

	MapKeyNoEquals().ValidateMap(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for valid keys, got: %s", resp.Diagnostics.Errors())
	}
}

func TestMapKeyNoEquals_KeyWithEquals(t *testing.T) {
	ctx := context.Background()

	mapVal, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"good-key":  "value1",
		"bad=key":   "value2",
		"other=bad": "value3",
	})
	if diags.HasError() {
		t.Fatalf("unexpected error creating map: %s", diags.Errors())
	}

	req := validator.MapRequest{
		Path:        path.Root("ssh_private_keys"),
		ConfigValue: mapVal,
	}
	resp := &validator.MapResponse{}

	MapKeyNoEquals().ValidateMap(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected errors for keys containing '=', got none")
	}

	// Should have errors for both bad keys
	errorCount := 0
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Invalid Map Key" {
			errorCount++
		}
	}
	if errorCount != 2 {
		t.Errorf("expected 2 validation errors, got %d", errorCount)
	}
}

func TestMapKeyNoEquals_NullMap(t *testing.T) {
	ctx := context.Background()

	req := validator.MapRequest{
		Path:        path.Root("ssh_private_keys"),
		ConfigValue: types.MapNull(types.StringType),
	}
	resp := &validator.MapResponse{}

	MapKeyNoEquals().ValidateMap(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for null map, got: %s", resp.Diagnostics.Errors())
	}
}

func TestMapKeyNoEquals_UnknownMap(t *testing.T) {
	ctx := context.Background()

	req := validator.MapRequest{
		Path:        path.Root("ssh_private_keys"),
		ConfigValue: types.MapUnknown(types.StringType),
	}
	resp := &validator.MapResponse{}

	MapKeyNoEquals().ValidateMap(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unknown map, got: %s", resp.Diagnostics.Errors())
	}
}

func TestMapKeyNoEquals_Description(t *testing.T) {
	ctx := context.Background()
	v := MapKeyNoEquals()

	desc := v.Description(ctx)
	if desc == "" {
		t.Error("expected non-empty description")
	}

	mdDesc := v.MarkdownDescription(ctx)
	if mdDesc == "" {
		t.Error("expected non-empty markdown description")
	}
}
