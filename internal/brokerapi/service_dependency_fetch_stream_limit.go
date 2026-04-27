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
	if r.seen >= r.maxBytes {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		if n > 0 {
			r.over = true
			return 0, dependencyResponseSizeLimitError{limitBytes: r.maxBytes}
		}
		return 0, err
	}

	remaining := r.maxBytes - r.seen
	if int64(len(p)) > remaining {
		p = p[:int(remaining)]
	}

	n, err := r.reader.Read(p)
	if n > 0 {
		r.seen += int64(n)
	}
	return n, err
}

func isStreamingSizeLimitError(err error) bool {
	var sizeErr dependencyResponseSizeLimitError
	return errors.As(err, &sizeErr)
}
