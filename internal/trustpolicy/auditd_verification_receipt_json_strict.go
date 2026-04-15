package trustpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func unmarshalJSONStrict(raw json.RawMessage, dest any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("unexpected trailing json tokens")
	}
	return nil
}
