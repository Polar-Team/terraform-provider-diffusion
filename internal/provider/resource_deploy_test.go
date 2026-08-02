package provider

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testPEM = "-----BEGIN OPENSSH PRIVATE KEY-----\nabc123\n-----END OPENSSH PRIVATE KEY-----\n"

func TestBase64EncodeMap_EncodesRawPEM(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	m, d := types.MapValueFrom(ctx, types.StringType, map[string]string{"default": testPEM})
	if d.HasError() {
		t.Fatalf("unexpected error building map: %s", d.Errors())
	}

	got := base64EncodeMap(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags.Errors())
	}

	want := base64.StdEncoding.EncodeToString([]byte(testPEM))
	if got["default"] != want {
		t.Errorf("expected %q, got %q", want, got["default"])
	}
	if got["default"] == testPEM {
		t.Error("value was not encoded")
	}
}

func TestBase64EncodeMap_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	m, d := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"deploy": "key-one",
		"backup": "key-two",
	})
	if d.HasError() {
		t.Fatalf("unexpected error building map: %s", d.Errors())
	}

	got := base64EncodeMap(ctx, m, &diags)
	if len(got) != 2 {
		t.Fatalf("expected 2 encoded keys, got %d (%v)", len(got), got)
	}
	if got["deploy"] != base64.StdEncoding.EncodeToString([]byte("key-one")) {
		t.Errorf("unexpected encoding for deploy: %q", got["deploy"])
	}
	if got["backup"] != base64.StdEncoding.EncodeToString([]byte("key-two")) {
		t.Errorf("unexpected encoding for backup: %q", got["backup"])
	}
}

func TestBase64EncodeMap_NullMapReturnsNil(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	if got := base64EncodeMap(ctx, types.MapNull(types.StringType), &diags); got != nil {
		t.Errorf("expected nil for null map, got %v", got)
	}
	if diags.HasError() {
		t.Errorf("expected no diagnostics, got: %s", diags.Errors())
	}
}

func TestBase64EncodeMap_UnknownMapReturnsNil(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	if got := base64EncodeMap(ctx, types.MapUnknown(types.StringType), &diags); got != nil {
		t.Errorf("expected nil for unknown map, got %v", got)
	}
	if diags.HasError() {
		t.Errorf("expected no diagnostics, got: %s", diags.Errors())
	}
}

func TestBase64EncodeMap_SkipsEmptyNullAndUnknownValues(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	m := types.MapValueMust(types.StringType, map[string]attr.Value{
		"set":     types.StringValue(testPEM),
		"empty":   types.StringValue(""),
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	})

	got := base64EncodeMap(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags.Errors())
	}
	if len(got) != 1 {
		t.Fatalf("expected only the populated key to be encoded, got %v", got)
	}
	if _, ok := got["set"]; !ok {
		t.Errorf("expected key %q to be present, got %v", "set", got)
	}
}

func TestBase64EncodeIfSet(t *testing.T) {
	if got := base64EncodeIfSet(""); got != "" {
		t.Errorf("expected empty string passthrough, got %q", got)
	}
	want := base64.StdEncoding.EncodeToString([]byte(testPEM))
	if got := base64EncodeIfSet(testPEM); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
