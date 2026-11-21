package workflow

import (
	_ "embed"

	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
	"github.com/alexfalkowski/go-service/v3/strings"
)

//go:embed prompts/common.txt
var commonInstructions string

//go:embed prompts/followup.txt
var followupInstructions string

func reviewInstructions(feature string, providers []provider.Strategy, followup bool) string {
	parts := make([]string, 0, 3+len(providers))
	parts = append(parts, feature, commonInstructions)
	if followup {
		parts = append(parts, followupInstructions)
	}
	for _, strategy := range providers {
		parts = append(parts, strategy.Guide())
	}

	return strings.Join("\n\n", parts...)
}
