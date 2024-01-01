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

func parseQueryParams(r *http.Request, values ...interface{}) error {

	params := ignoreEmptyParams(r.URL.Query())

	limitParams := make(map[string][]string)
	filterParams := make(map[string][]string)

	limitKeys := make(map[string]struct{})
	for _, key := range getSchemaTags(types.Limit{}) {
		limitKeys[key] = struct{}{}
	}

	for _, value := range values {
		for key, val := range params {
			if _, ok := limitKeys[key]; ok {
				limitParams[key] = val
			} else {
				filterParams[key] = val
			}
		}

		param := filterParams
		if _, ok := value.(*types.Limit); ok {
			param = limitParams
		}

		if err := decoder.Decode(value, param); err != nil {
			return fmt.Errorf("failed to decode params: %w", err)
		}
	}

	return nil
}

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
