// Package graphql for grid graphql support
package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// GraphQl for tf graphql
type GraphQl struct {
	url string
}

// NewGraphQl new tf graphql
func NewGraphQl(url string) (GraphQl, error) {
	return GraphQl{url}, nil
}

// GetItemTotalCount return count of items
func (g *GraphQl) GetItemTotalCount(itemName string, options string) (float64, error) {
	countBody := fmt.Sprintf(`query { items: %vConnection%v { count: totalCount } }`, itemName, options)
	requestBody := map[string]interface{}{"query": countBody}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	bodyReader := bytes.NewReader(jsonBody)

	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	countResponse, err := cl.Post(g.url, "application/json", bodyReader)
	if err != nil {
		return 0, err
	}

	queryData, err := parseHTTPResponse(countResponse)
	if err != nil {
		return 0, err
	}

	countMap := queryData["data"].(map[string]interface{})
	countItems := countMap["items"].(map[string]interface{})
	count := countItems["count"].(float64)

	return count, nil
}

// Query queries graphql
func (g *GraphQl) Query(body string, variables map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	requestBody := map[string]interface{}{"query": body, "variables": variables}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return result, err
	}

	bodyReader := bytes.NewReader(jsonBody)

	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := cl.Post(g.url, "application/json", bodyReader)
	if err != nil {
		return result, err
	}

	queryData, err := parseHTTPResponse(resp)
	if err != nil {
		return result, err
	}

	result = queryData["data"].(map[string]interface{})
	return result, nil
}

func parseHTTPResponse(resp *http.Response) (map[string]interface{}, error) {
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{}, err
	}

	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.Unmarshal(resBody, &data)
	if err != nil {
		return map[string]interface{}{}, err
	}

	if resp.StatusCode >= 400 {
		return map[string]interface{}{}, errors.Errorf("request failed with status code: %d with error %v", resp.StatusCode, data)
	}

	return data, nil
}
