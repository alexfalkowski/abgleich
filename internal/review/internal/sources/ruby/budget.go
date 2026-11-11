package ruby

import (
	"github.com/alexfalkowski/go-service/v2/context"
	"github.com/alexfalkowski/go-service/v2/errors"
	"github.com/alexfalkowski/go-service/v2/io"
)

// One budget covers Ruby downloads and archive reads during a review, including
// compressed input, expanded bytes, and rescans. It bounds retained archives and source
// as well as decompression work without a separate file-count or excerpt limit.
const gemByteLimit = 32 << 20

var errGemByteLimit = errors.New("review: Ruby gem byte budget exceeded")

// gemReader checks cancellation and probes one byte past a cumulative limit so
// reaching the boundary is not mistaken for a valid end of stream.
type gemReader struct {
	ctx    context.Context
	reader io.Reader
	used   *int64
}

func (r *gemReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if *r.used > gemByteLimit {
		return 0, errGemByteLimit
	}

	buffer = buffer[:min(int64(len(buffer)), gemByteLimit-*r.used+1)]
	n, err := r.reader.Read(buffer)
	*r.used += int64(n)
	if *r.used > gemByteLimit {
		return 0, errGemByteLimit
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return n, err
}
