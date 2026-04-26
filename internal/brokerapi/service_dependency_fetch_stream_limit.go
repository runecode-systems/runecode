package brokerapi

import (
	"errors"
	"fmt"
	"io"
)

type dependencyResponseSizeLimitError struct {
	limitBytes int64
}

func (e dependencyResponseSizeLimitError) Error() string {
	return fmt.Sprintf("dependency fetch response exceeded allowed max_response_bytes (%d)", e.limitBytes)
}

func newStreamingSizeLimitReader(reader io.Reader, maxBytes int64) io.Reader {
	if maxBytes <= 0 {
		return reader
	}
	return &streamingSizeLimitReader{reader: reader, maxBytes: maxBytes}
}

type streamingSizeLimitReader struct {
	reader   io.Reader
	maxBytes int64
	seen     int64
	over     bool
}

func (r *streamingSizeLimitReader) Read(p []byte) (int, error) {
	if r.over {
		return 0, dependencyResponseSizeLimitError{limitBytes: r.maxBytes}
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.seen += int64(n)
		if r.seen > r.maxBytes {
			r.over = true
			return n, dependencyResponseSizeLimitError{limitBytes: r.maxBytes}
		}
	}
	return n, err
}

func isStreamingSizeLimitError(err error) bool {
	var sizeErr dependencyResponseSizeLimitError
	return errors.As(err, &sizeErr)
}
