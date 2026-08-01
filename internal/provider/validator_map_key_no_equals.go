package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// mapKeyNoEquals validates that map keys do not contain the '=' character.
// This is required because the CLI argument format uses '=' as a delimiter
// (e.g., --ssh-key-base64 name=base64value), so keys containing '=' would
// break parsing on the consumer side.
type mapKeyNoEquals struct{}

func (v mapKeyNoEquals) Description(_ context.Context) string {
	return "map keys must not contain the '=' character"
}

func (v mapKeyNoEquals) MarkdownDescription(_ context.Context) string {
	return "map keys must not contain the `=` character"
}

func (v mapKeyNoEquals) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for key := range req.ConfigValue.Elements() {
		if strings.Contains(key, "=") {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Map Key",
				fmt.Sprintf("Map key %q contains '=' which is not allowed. The '=' character is used as a delimiter in the CLI argument format.", key),
			)
		}
	}
}

// MapKeyNoEquals returns a validator that ensures no map key contains '='.
func MapKeyNoEquals() validator.Map {
	return mapKeyNoEquals{}
}
