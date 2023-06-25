package explorer

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/mw"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func errorReply(err error) mw.Response {
	if errors.Is(err, ErrNodeNotFound) {
		return mw.NotFound(err)
	} else if errors.Is(err, ErrGatewayNotFound) {
		return mw.NotFound(err)
	} else if errors.Is(err, ErrBadGateway) {
		return mw.BadGateway(err)
	} else {
		return mw.Error(err)
	}
}

func getLimit(r *http.Request) (types.Limit, error) {
	var limit types.Limit

	page := r.URL.Query().Get("page")
	size := r.URL.Query().Get("size")
	retCount := r.URL.Query().Get("ret_count")
	randomize := r.URL.Query().Get("randomize")
	if page == "" {
		page = "1"
	}
	if size == "" {
		size = "50"
	}
	parsed, err := strconv.ParseUint(page, 10, 64)
	if err != nil {
		return limit, errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse page %s", err.Error()))
	}
	limit.Page = parsed

	parsed, err = strconv.ParseUint(size, 10, 64)
	if err != nil {
		return limit, errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse size %s", err.Error()))
	}
	limit.Size = parsed

	limit.RetCount = false
	if retCount == "true" {
		limit.RetCount = true
	}

	limit.Randomize = false
	if randomize == "true" {
		limit.Randomize = true
	}

	// TODO: readd the check once clients are updated
	// if limit.Size > maxPageSize {
	// 	return limit, errors.Wrapf(ErrBadRequest, "max page size is %d", maxPageSize)
	// }
	return limit, nil
}

func extractRequestParams(r *http.Request, filter interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(filter))

	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Tag.Get("json")
		fieldType := v.Field(i).Type().String()

		value := r.URL.Query().Get(name)
		if value != "" {
			switch fieldType {
			case "*uint64":
				parsed, err := strconv.ParseUint(value, 10, 64)
				if err != nil {
					return errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse %s %s", name, err.Error()))
				}

				v.Field(i).Set(reflect.ValueOf(&parsed))

			case "*uint32":
				parsed, err := strconv.ParseUint(value, 10, 32)
				if err != nil {
					return errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse %s %s", name, err.Error()))
				}

				castUint32 := uint32(parsed)
				v.Field(i).Set(reflect.ValueOf(&castUint32))

			case "*string":
				v.Field(i).Set(reflect.ValueOf(&value))

			case "*bool":
				trueVal := true
				falseVal := false

				if value == "true" {
					v.Field(i).Set(reflect.ValueOf(&trueVal))
				}

				if value == "false" {
					v.Field(i).Set(reflect.ValueOf(&falseVal))
				}

			case "[]uint64":
				strList := strings.Split(value, ",")
				var list []uint64
				for _, str := range strList {
					parsed, err := strconv.ParseUint(str, 10, 64)
					if err != nil {
						return errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse %s %s", name, err.Error()))
					}

					list = append(list, parsed)
				}

				v.Field(i).Set(reflect.ValueOf(list))
			case "[]uint32":
				strList := strings.Split(value, ",")
				var list []uint32
				for _, str := range strList {
					parsed, err := strconv.ParseUint(str, 10, 32)
					if err != nil {
						return errors.Wrap(ErrBadRequest, fmt.Sprintf("couldn't parse %s %s", name, err.Error()))
					}

					list = append(list, uint32(parsed))
				}

				v.Field(i).Set(reflect.ValueOf(list))
			default:
				return errors.Wrapf(ErrBadGateway, "failed to parse type %s for field %s", fieldType, name)
			}

		}
	}

	return nil
}

// test nodes?status=up&free_ips=0&free_cru=1&free_mru=1&free_hru=1&country=Belgium&city=Unknown&ipv4=true&ipv6=true&domain=false
// handleNodeRequestsQueryParams takes the request and restore the query paramas, handle errors and set default values if not available
func (a *App) handleNodeRequestsQueryParams(r *http.Request) (types.NodeFilter, types.Limit, error) {
	var filter types.NodeFilter
	var limit types.Limit

	if err := extractRequestParams(r, &filter); err != nil {
		return filter, limit, err
	}

	limit, err := getLimit(r)
	if err != nil {
		return filter, limit, err
	}
	trueval := true
	if strings.HasSuffix(r.URL.Path, "gateways") {
		filter.Domain = &trueval
		filter.IPv4 = &trueval
	}
	return filter, limit, nil
}

// test farms?free_ips=1&pricing_policy_id=1&version=4&farm_id=23&twin_id=291&name=Farm-1&stellar_address=13VrxhaBZh87ZP8nuYF4LtAhnDPWMfSrMUvHeRAFaqN43W1X
// handleFarmRequestsQueryParams takes the request and restore the query paramas, handle errors and set default values if not available
func (a *App) handleFarmRequestsQueryParams(r *http.Request) (types.FarmFilter, types.Limit, error) {
	var filter types.FarmFilter
	var limit types.Limit

	if err := extractRequestParams(r, &filter); err != nil {
		return filter, limit, err
	}

	limit, err := getLimit(r)
	if err != nil {
		return filter, limit, err
	}
	return filter, limit, nil
}

// test twins?twin_id=7
// handleTwinRequestsQueryParams takes the request and restore the query paramas, handle errors and set default values if not available
func (a *App) handleTwinRequestsQueryParams(r *http.Request) (types.TwinFilter, types.Limit, error) {
	var filter types.TwinFilter
	var limit types.Limit

	if err := extractRequestParams(r, &filter); err != nil {
		return filter, limit, err
	}

	limit, err := getLimit(r)
	if err != nil {
		return filter, limit, err
	}
	return filter, limit, nil
}

// test contracts?contract_id=7
// HandleContractRequestsQueryParams takes the request and restore the query paramas, handle errors and set default values if not available
func (a *App) handleContractRequestsQueryParams(r *http.Request) (types.ContractFilter, types.Limit, error) {
	var filter types.ContractFilter
	var limit types.Limit

	if err := extractRequestParams(r, &filter); err != nil {
		return filter, limit, err
	}

	limit, err := getLimit(r)
	if err != nil {
		return filter, limit, err
	}
	return filter, limit, nil
}

// test stats?status=up
// HandleNodeRequestsQueryParams takes the request and restore the query paramas, handle errors and set default values if not available
func (a *App) handleStatsRequestsQueryParams(r *http.Request) (types.StatsFilter, error) {
	var filter types.StatsFilter

	if err := extractRequestParams(r, &filter); err != nil {
		return filter, err
	}

	return filter, nil
}

// getNodeData is a helper function that wraps fetch node data
// it caches the results in redis to save time
func (a *App) getNodeData(nodeIDStr string) (types.NodeWithNestedCapacity, error) {
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		return types.NodeWithNestedCapacity{}, errors.Wrap(ErrBadGateway, fmt.Sprintf("invalid node id %d: %s", nodeID, err.Error()))
	}
	info, err := a.db.GetNode(uint32(nodeID))
	if errors.Is(err, db.ErrNodeNotFound) {
		return types.NodeWithNestedCapacity{}, ErrNodeNotFound
	} else if err != nil {
		// TODO: wrapping
		return types.NodeWithNestedCapacity{}, err
	}
	apiNode := nodeWithNestedCapacityFromDBNode(info)
	return apiNode, nil
}
