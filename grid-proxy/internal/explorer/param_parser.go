package explorer

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/gorilla/schema"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var decoder *schema.Decoder

func init() {
	decoder = schema.NewDecoder()
}

func parseQueryParams(r *http.Request, types_ ...interface{}) error {
	params := ignoreEmptyParams(r.URL.Query())

	limitParams := make(map[string][]string)
	filterParams := make(map[string][]string)

	// a helper step to be able to decide if a param is limit query or a filter
	limitKeys := make(map[string]struct{})
	for _, key := range getSchemaTags(types.Limit{}) {
		limitKeys[key] = struct{}{}
	}

	for _, type_ := range types_ {
		// separate the values of filter/limit into two maps so it is clear what we decode in this iteration
		// not doing this will make the decoding always fails cause r.URL.Query slice will not fit in only filter or limit
		// but it has values for both
		for key, val := range params {
			if _, ok := limitKeys[key]; ok {
				limitParams[key] = val
			} else {
				filterParams[key] = val
			}
		}

		// deciding which param values will be decoded in the iteration
		// default it will be the filter map (for nodes/farms/etc..)
		// but if the interface is inferable to `Limit` type then it is limit
		param := filterParams
		if _, ok := type_.(*types.Limit); ok {
			param = limitParams
		}

		// the decoder will decode the map of values from param variable to the related passed object
		if err := decoder.Decode(type_, param); err != nil {
			return fmt.Errorf("failed to decode params: %w", err)
		}
	}

	return nil
}

// returns a slice of all the possible params for this type
func getSchemaTags(type_ interface{}) []string {
	t := reflect.TypeOf(type_)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var tags []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("schema")
		if tag != "" {
			name := strings.Split(tag, ",")[0]
			tags = append(tags, name)
		}
	}

	return tags
}

// returns a map of params that have a value
func ignoreEmptyParams(params url.Values) url.Values {
	newParams := make(url.Values)
	for key, val := range params {
		for _, v := range val {
			if v != string("") {
				newParams[key] = append(newParams[key], v)
			}
		}
	}

	return newParams
}
