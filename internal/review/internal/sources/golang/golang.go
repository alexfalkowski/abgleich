package golang

import "github.com/alexfalkowski/abgleich/internal/review/internal/sources/provider"

type dependencyRequest struct {
	Manifest string `json:"manifest"`
	Module   string `json:"module"`
	Path     string `json:"path"`

	provider.Range
}

type referenceRequest struct {
	Manifest string `json:"manifest"`
}
