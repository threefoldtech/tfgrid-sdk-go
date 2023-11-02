package explorer

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func TestZeroFilterDecode(t *testing.T) {
	filters := []interface{}{
		types.NodeFilter{},
		types.TwinFilter{},
		types.TwinFilter{},
		types.FarmFilter{},
		types.ContractFilter{},
		types.StatsFilter{},
		types.Limit{},
	}

	for _, filter := range filters {
		testDecode(t, filter)
	}
}

func testDecode(t *testing.T, filter interface{}) {
	req, err := http.NewRequest(http.MethodGet, "gridproxy.com", nil)
	assert.NoError(t, err)

	want := reflect.New(reflect.TypeOf(filter)).Elem()

	urlValues := url.Values{}
	for i := 0; i < want.NumField(); i++ {
		var x, y reflect.Value
		if want.Field(i).Type().Kind() == reflect.Pointer {
			x = reflect.New(want.Field(i).Type().Elem())
		} else {
			x = reflect.New(want.Field(i).Type())
		}
		y = x.Elem()

		want.Field(i).Set(x)

		schemaTag, ok := want.Type().Field(i).Tag.Lookup("schema")
		if !ok {
			t.Fatalf("schema tag must be provided for all fields in a filter")
		}

		name := strings.Split(schemaTag, ",")[0]
		if want.Field(i).Type().Kind() == reflect.Slice {
			for j := 0; j < want.Field(i).Len(); j++ {
				urlValues.Add(name, fmt.Sprintf("%v", want.Field(i).Index(j).Interface()))
			}
		} else {
			urlValues.Add(name, fmt.Sprintf("%v", y.Interface()))
		}

	}

	req.URL.RawQuery = urlValues.Encode()

	zeroFilter := reflect.New(reflect.TypeOf(filter))
	got := zeroFilter.Interface()

	err = parseQueryParams(req, got)
	assert.NoError(t, err)

	assert.Equal(t, want.Interface(), reflect.ValueOf(got).Elem().Interface())
}
