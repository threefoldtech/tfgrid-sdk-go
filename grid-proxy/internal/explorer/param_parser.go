package explorer

import (
	"net/http"

	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

func parseQueryParams(r *http.Request, values ...interface{}) error {
	params := r.URL.Query()

	// ignore the empty params
	for key, val := range params {
		for _, v := range val {
			if v == string("") {
				delete(params, key)
			}
		}
	}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)

	for _, value := range values {
		if err := decoder.Decode(value, params); err != nil {
			return errors.Wrapf(err, "failed to decode %s parameter", value)
		}
	}

	return nil
}
