package workflow

// listSourceFilesRequest decodes the model's literal path filter for repository discovery.
type listSourceFilesRequest struct {
	Filter string `json:"filter"`
}

// readSourcesRequest decodes one or more repository paths for one authorized source lookup.
type readSourcesRequest struct {
	Paths []string `json:"paths"`
}

// goReferenceRequest selects the reviewed go.mod that determines the Go release notes.
type goReferenceRequest struct {
	Manifest string `json:"manifest"`
}

// dependencyRequest selects a declared module and file or directory using the reviewed manifest.
type dependencyRequest struct {
	Manifest string `json:"manifest"`
	Module   string `json:"module"`
	Path     string `json:"path"`
}
