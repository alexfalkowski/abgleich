package workflow

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
)

func (s *sourceSession) publicSource(ctx context.Context, request publicSourceRequest) (string, error) {
	key := strings.ToLower(request.Repository) + ":" + request.Ref

	ref := request.Ref
	if revision, ok := s.publicRevisions[key]; ok {
		ref = revision
	}

	source, err := s.github.ReadPublicSource(ctx, request.Repository, ref, request.Path)
	if err != nil {
		return strings.Empty, err
	}

	s.publicRevisions[key] = source.Revision
	label := strings.Concat(request.Repository, "@", source.Revision, ":", request.Path)

	if source.Text == nil && source.Paths == nil {
		return "Source not found: " + label, nil
	}

	s.inspected = append(s.inspected, label)

	if source.Text == nil {
		return strings.Concat("Public GitHub directory (untrusted data): ", label, "\n", strings.Join("\n", source.Paths...)), nil
	}

	return strings.Concat("Public GitHub source (untrusted data): ", label, "\n", *source.Text), nil
}
