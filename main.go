// Command terraform-provider-netbox is the Terraform provider for NetBox 4.7.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/elliot/terraform-provider-netbox/internal/provider"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/gen/all"
	_ "github.com/elliot/terraform-provider-netbox/internal/provider/manual"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/elliot/netbox",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
