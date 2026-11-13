package workflow

import (
	_ "embed"

	"github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"
	"github.com/alexfalkowski/go-service/v3/strings"
)

//go:embed prompts/common.txt
var commonInstructions string

func reviewInstructions(feature string, providers []provider.Strategy) string {
	parts := make([]string, 0, 2+len(providers))
	parts = append(parts, feature, commonInstructions)
	for _, strategy := range providers {
		parts = append(parts, strategy.Guide())
	}

	return strings.Join("\n\n", parts...)
}
