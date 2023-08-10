package explorer

import (
	"net/http"

	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

func parseQueryParams(r *http.Request, values ...interface{}) error {
	params := r.URL.Query()

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)

	for idx := range values {
		if err := decoder.Decode(values[idx], params); err != nil {
			return errors.Wrap(err, "failed to decode filter parameters")
		}
	}

	return nil
}
