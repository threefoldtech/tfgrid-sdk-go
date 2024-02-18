package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer/types"
	// "rmbClient/peer"
)

const (
	chainUrl = "wss://tfchain.grid.tf/"
	relayUrl = "ws://localhost:"
	mnemonic = "<mnemonic>"
)

type Node struct {
	TwinId uint32 `json:"twinId"`
}

var static = []uint32{7, 9, 10, 13, 14, 16, 22, 23, 24, 27, 29, 35, 46, 47, 69, 82, 86, 164, 189, 209, 222, 242, 346, 399, 400, 401, 403, 405, 408, 410, 422, 420, 416, 421, 423, 417, 415, 419, 425, 429, 433, 568, 569, 571, 587, 605, 608, 625, 628, 639, 661, 687, 691, 702, 703, 753, 754, 781, 798, 809, 810, 812, 813, 816, 851, 854, 872, 877, 878, 880, 924, 927, 928, 929, 931, 937, 938, 940, 941, 942, 943, 945, 946, 949, 952, 956, 957, 970, 971, 989, 1021, 1025, 1034, 1038, 1041, 1042, 1043, 1044, 1050, 1058, 1062, 1067, 1071, 1083, 1091, 1114, 1125, 1142, 1143, 1149, 1150, 1151, 1162, 1182, 1301, 1388, 1461, 1510, 1500, 1545, 1553, 1582, 1591, 1633, 1635, 1682, 1741, 1747, 1779, 1826, 1858, 1860, 1861, 1863, 1865, 1886, 1889, 1933, 1936, 1937, 1985, 1987, 1988, 2022, 2029, 2030, 2036, 2060, 2078, 2086, 2089, 2090, 2092, 2095, 2099, 2100, 2101, 2103, 2104, 2105, 2107, 2111, 2114, 2115, 2118, 2119, 2122, 2124, 2126, 2127, 2131, 2133, 2132, 2135, 2137, 2143, 2145, 2150, 2152, 2154, 2155, 2158, 2163, 2166, 2164, 2165, 2167, 2168, 2170, 2171, 2173, 2177, 2180, 2182, 2186, 2188, 2187, 2191, 2190, 2192, 2194, 2195, 2199, 2196, 2198, 2197, 2200, 2204, 2207, 2208, 2210, 2209, 2212, 2214, 2213, 2216, 2215, 2218, 2219, 2223, 2225, 2227, 2230, 2231, 2234, 2236, 2235, 2238, 2237, 2239, 2242, 2241, 2240, 2246, 2248, 2247, 2250, 2249, 2252, 2251, 2253, 2254, 2255, 2261, 2260, 2264, 2263, 2265, 2272, 2268, 2270, 2273, 2274, 2277, 2279, 2281, 2284, 2285, 2286, 2287, 2288, 2289, 2290, 2295, 2296, 2297, 2302, 2303, 2304, 2305, 2307, 2308, 2311, 2310, 2314, 2317, 2320, 2318, 2322, 2324, 2325, 2327, 2326, 2331, 2332, 2334, 2333, 2336, 2339, 2343, 2344, 2349, 2347, 2352, 2353, 2355, 2356, 2357, 2358, 2361, 2362, 2365, 2368, 2369, 2370, 2371, 2372, 2374, 2375, 2376, 2377, 2378, 2379, 2381, 2382, 2384, 2395, 2397, 2398, 2399, 2402, 2401, 2403, 2405, 2404, 2408, 2410, 2409, 2411, 2412, 2413, 2415, 2414, 2417, 2416, 2420, 2424, 2429, 2432, 2434, 2435, 2438, 2442, 2446, 2445, 2448, 2451, 2449, 2450, 2458, 2459, 2463, 2465, 2471, 2473, 2475, 2478, 2481, 2484, 2485, 2486, 2487, 2488, 2489, 2490, 2492, 2496, 2493, 2499, 2502, 2503, 2505, 2506, 2507, 2511, 2510, 2509, 2513, 2512, 2514, 2518, 2520, 2519, 2522, 2524, 2523, 2528, 2525, 2536, 2537, 2539, 2538, 2541, 2543, 2545, 2548, 2550, 2549, 2553, 2552, 2554, 2556, 2566, 2573, 2574, 2578, 2580, 2579, 2581, 2586, 2587, 2588, 2595, 2596, 2597, 2605, 2615, 2616, 2623, 2626, 2634, 2635, 2637, 2644, 2648, 2652, 2656, 2664, 2666, 2667, 2669, 2670, 2673, 2676, 2695, 2699, 2700, 2704, 2705, 2706, 2707, 2710, 2712, 2713, 2716, 2723, 2733, 2735, 2737, 2738, 2740, 2741, 2751, 2763, 2776, 2781, 2784, 2787, 2789, 2796, 2800, 2806, 2812, 2815, 2831, 2832, 2836, 2837, 2847, 2848, 2851, 2860, 2866, 2867, 2869, 2870, 2871, 2874, 2875, 2879, 2891, 2893, 2918, 2920, 2924, 2926, 2932, 2933, 2934, 2935, 2936, 2937, 2938, 2939, 2940, 2942, 2944, 2945, 2947, 2949}

const use_static = true

func main() {
	subMan := substrate.NewManager(chainUrl)

	count := 500
	var wg sync.WaitGroup
	wg.Add(count)

	received := 0
	handler := func(ctx context.Context, peer peer.Peer, env *types.Envelope, err error) {
		received += 1
		log.Info().Int("received", received).Msg("received responses so far")
		defer wg.Done()

		if err != nil {
			log.Error().Err(err).Uint32("twin", env.Source.Twin).Msg("twin error")
			return
		}

		var result = env.GetPlain()
		var version string
		if err := json.Unmarshal(result, &version); err != nil {
			log.Error().Err(err).Msg("failed to decode response")
			return
		}

		log.Info().Uint32("twin", env.Source.Twin).Str("version", version).Msg("received response")
	}

	bus, err := peer.NewPeer(context.Background(),
		mnemonic,
		subMan,
		handler,
		peer.WithKeyType(peer.KeyTypeSr25519),
		peer.WithSession("rmb-playground999"),
		peer.WithTwinCache(10*60*60), // in seconds that's 10 hours
	)

	if err != nil {
		fmt.Println("failed to create peer client: %w", err)
		os.Exit(1)
	}

	var twins []uint32
	if use_static {
		twins = static
	} else {

		res, err := http.Get(fmt.Sprintf("https://gridproxy.bknd1.ninja.tf/nodes?healthy=true&size=%d", count))
		if err != nil {
			fmt.Println("failed getting nodes")
		}

		var nodes []Node
		err = json.NewDecoder(res.Body).Decode(&nodes)
		if err != nil {
			fmt.Println("failed to decode res")
		}

		twins = Map(nodes, func(node Node) uint32 { return node.TwinId })
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	for _, twin := range twins {

		if err := bus.SendRequest(ctx, fmt.Sprintf("pinging-%d", twin), twin, nil, "rmb.version", nil); err != nil {
			log.Error().Err(err).Uint32("twin", twin).Msg("failed to send message request to twin")
		}

	}

	log.Info().Msg("ALL MESSAGES HAVE BEEN SENT #############################################################################")
	wg.Wait()

}

func rmbCall(ctx context.Context, bus *peer.RpcClient, twinId uint32) error {

	var res interface{}
	err := bus.Call(ctx, twinId, "rmb.version", nil, &res)
	if err != nil {
		return err
	}

	log.Info().Uint32("twinId", twinId).Msgf("%+v", res)

	return nil
}

func Map[SA []A, SB []B, A any, B any](s SA, fn func(A) B) SB {

	d := make(SB, 0, len(s))
	for _, a := range s {
		d = append(d, fn(a))
	}

	return d
}
