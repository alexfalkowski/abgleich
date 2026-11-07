package github

import (
	"github.com/alexfalkowski/go-service/v2/cache"
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/strings"
	"github.com/alexfalkowski/go-service/v2/time"
)

const (
	cacheKind      = "ttlcache"
	sourceCacheTTL = 15 * time.Minute
)

type cachedSourceFile struct {
	Content string `json:"content"`
	Binary  bool   `json:"binary"`
}

type cachingService struct {
	Service
}

// ListSourceFiles reuses validated trees by repository identity and revision.
// Cache misses or failures fall back to the wrapped Service; successful reads
// are retained when the cache accepts them.
func (s *cachingService) ListSourceFiles(ctx context.Context, repo *Repository, revision string) ([]*TreeEntry, error) {
	if err := validateSourceRequest(repo, revision, nil); err != nil {
		return nil, err
	}

	key := strings.Concat("review-source:", repo.GetNodeID(), ":tree:", revision)
	if entries, ok, _ := cache.Get[[]*TreeEntry](ctx, key); ok {
		return *entries, ctx.Err()
	}

	entries, err := s.Service.ListSourceFiles(ctx, repo, revision)
	if err == nil {
		_ = cache.Persist(ctx, key, &entries, sourceCacheTTL)
	}
	return entries, err
}

// ReadSourceFiles reuses validated text and binary exclusions by repository
// identity and blob SHA. Cache misses are fetched together from the wrapped
// Service; cache failures do not prevent source reads.
func (s *cachingService) ReadSourceFiles(ctx context.Context, repo *Repository, revision string, files []*TreeEntry) (map[string]string, error) {
	if err := validateSourceRequest(repo, revision, files); err != nil {
		return nil, err
	}

	prefix := strings.Concat("review-source:", repo.GetNodeID(), ":blob:")
	contents := make(map[string]string, len(files))

	var missing []*TreeEntry
	for _, file := range files {
		source, ok, _ := cache.Get[cachedSourceFile](ctx, prefix+file.GetSHA())
		if !ok {
			missing = append(missing, file)
			continue
		}
		if !source.Binary {
			contents[file.GetPath()] = source.Content
		}
	}
	if len(missing) == 0 {
		return contents, ctx.Err()
	}

	current, err := s.Service.ReadSourceFiles(ctx, repo, revision, missing)
	if err != nil {
		return nil, err
	}

	for _, file := range missing {
		content, ok := current[file.GetPath()]
		if ok {
			contents[file.GetPath()] = content
		}

		// Cache failures, including oversized values, must never limit source reads.
		_ = cache.Persist(ctx, prefix+file.GetSHA(), &cachedSourceFile{Content: content, Binary: !ok}, sourceCacheTTL)
	}

	return contents, ctx.Err()
}
