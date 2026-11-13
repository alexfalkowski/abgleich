package cmd

import (
	"github.com/alexfalkowski/go-service/v3/cli"
	"github.com/alexfalkowski/go-service/v3/strings"
)

// RegisterServer for cmd.
func RegisterServer(command cli.Commander) {
	cmd := command.AddServer("server", "Start abgleich server", Module)
	cmd.AddConfig(strings.Empty)
}
