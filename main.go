// Command terraform-provider-diffusion is a Terraform/OpenTofu provider that
// bridges Terraform infrastructure data into Ansible deployment workflows
// managed by diffusion.
//
// The provider exposes:
//   - diffusion_deploy    – deploys Ansible roles to remote hosts
//   - diffusion_inventory – (data source) renders an Ansible YAML inventory
//
// The provider shells out to the `diffusion deploy` CLI command, so the
// diffusion binary must be installed and available in $PATH (or configured
// via the provider's diffusion_binary attribute).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/Polar-Team/terraform-provider-diffusion/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable provider debug mode (for use with delve)")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/Polar-Team/diffusion",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(Version), opts)
	if err != nil {
		log.Fatal(err)
	}
}
