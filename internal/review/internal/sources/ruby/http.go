package ruby

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
	"github.com/alexfalkowski/go-service/v2/net/http"
)

func (s *Provider) readSource(ctx context.Context, endpoint string, used *int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Prefix("review: source request", err)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("review: public source unavailable")
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("review: RubyGems rate limited")
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("review: public source unavailable")
	}
	if response.ContentLength > gemByteLimit-*used {
		return nil, errGemByteLimit
	}

	reader := &gemReader{ctx: ctx, reader: response.Body, used: used}
	content, _, err := io.ReadAll(reader)
	return content, errors.Prefix("review: read public source", err)
}
