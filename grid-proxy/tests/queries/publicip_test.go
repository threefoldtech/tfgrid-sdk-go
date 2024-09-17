package test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
)

func TestPublicIP(t *testing.T) {
	t.Parallel()

	t.Run("public_ip pagination test", func(t *testing.T) {
		t.Parallel()

		f := types.PublicIpFilter{}
		l := types.Limit{
			Size:     100,
			Page:     1,
			RetCount: true,
		}

		for {
			wantRes, wantCount, err := mockClient.PublicIps(context.Background(), f, l)
			require.NoError(t, err)

			gotRes, gotCount, err := gridProxyClient.PublicIps(context.Background(), f, l)
			require.NoError(t, err)

			require.Equal(t, wantCount, gotCount)

			require.True(t, reflect.DeepEqual(wantRes, gotRes), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(wantRes, gotRes)))

			if l.Page*l.Size >= uint64(wantCount) {
				break
			}
			l.Page++
		}
	})

	t.Run("public_ip filtration test", func(t *testing.T) {
		f := types.PublicIpFilter{}
		fv := reflect.ValueOf(&f).Elem()
		l := types.Limit{
			Size:     9999999,
			Page:     1,
			RetCount: true,
		}
		agg := calcPublicIpsAggregates(&data)

		// for each field on the filters struct
		for i := 0; i < fv.NumField(); i++ {
			generator, ok := publicIpGen[fv.Type().Field(i).Name]
			require.True(t, ok, "Filter field %s has no random value generator", fv.Type().Field(i).Name)

			// set the generated random values for the empty filter filed
			randomFieldValue := generator(agg)
			if randomFieldValue == nil {
				continue
			}
			if fv.Field(i).Type().Kind() != reflect.Slice {
				fv.Field(i).Set(reflect.New(fv.Field(i).Type().Elem()))
			}
			fv.Field(i).Set(reflect.ValueOf(randomFieldValue))

			// compare the clients responses
			want, wantCount, err := mockClient.PublicIps(context.Background(), f, l)
			require.NoError(t, err)

			got, gotCount, err := gridProxyClient.PublicIps(context.Background(), f, l)
			require.NoError(t, err)

			require.Equal(t, wantCount, gotCount)

			require.True(t, reflect.DeepEqual(want, got), fmt.Sprintf("Used Filter:\n%s", SerializeFilter(f)), fmt.Sprintf("Difference:\n%s", cmp.Diff(want, got)))

			fv.Field(i).Set(reflect.Zero(fv.Field(i).Type()))
		}
	})
}

type PublicIpsAggregate struct {
	farmIds []uint64
	ips     []string
	gateway []string
}

func calcPublicIpsAggregates(data *mock.DBData) (agg PublicIpsAggregate) {
	for _, ip := range data.PublicIPs {
		agg.farmIds = append(agg.farmIds, data.FarmIDMap[ip.FarmID])
		agg.ips = append(agg.ips, ip.IP)
		agg.gateway = append(agg.gateway, ip.Gateway)
	}
	return
}

var publicIpGen = map[string]func(agg PublicIpsAggregate) any{
	"Free": func(_ PublicIpsAggregate) any {
		v := true
		if flip(.5) {
			v = false
		}
		return &v
	},
	"FarmIDs": func(agg PublicIpsAggregate) any {
		randomLen := rand.Intn(5)
		return getRandomSliceFrom(agg.farmIds, randomLen)
	},
	"Ip": func(agg PublicIpsAggregate) any {
		return &agg.ips[rand.Intn(len(agg.ips))]
	},
	"Gateway": func(agg PublicIpsAggregate) any {
		return &agg.gateway[rand.Intn(len(agg.gateway))]
	},
}
