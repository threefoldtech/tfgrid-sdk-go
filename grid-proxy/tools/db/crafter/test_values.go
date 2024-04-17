package crafter

import "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"

var (
	countries = []string{"Belgium", "United States", "Egypt", "United Kingdom"}
	regions   = map[string]string{
		"Belgium":        "Europe",
		"United States":  "Americas",
		"Egypt":          "Africa",
		"United Kingdom": "Europe",
	}
	countriesCodes = map[string]string{
		"Belgium":        "BG",
		"United States":  "US",
		"Egypt":          "EG",
		"United Kingdom": "UK",
	}
	cities = map[string][]string{
		"Belgium":        {"Brussels", "Antwerp", "Ghent", "Charleroi"},
		"United States":  {"New York", "Chicago", "Los Angeles", "San Francisco"},
		"Egypt":          {"Cairo", "Giza", "October", "Nasr City"},
		"United Kingdom": {"London", "Liverpool", "Manchester", "Cambridge"},
	}
	bios = []types.BIOS{
		{Vendor: "SeaBIOS", Version: "Arch Linux 1.16.3-1-1"},
		{Vendor: "American Megatrends Inc.", Version: "3.2"},
		{Vendor: "American Megatrends Inc.", Version: "F4"},
		{Vendor: "American Megatrends Inc.", Version: "P3.60"},
	}

	baseboard = []types.Baseboard{
		{Manufacturer: "Supermicro", ProductName: "X9DRi-LN4+/X9DR3-LN4+"},
		{Manufacturer: "GIGABYTE", ProductName: "MCMLUEB-00"},
		{Manufacturer: "INTEL Corporation", ProductName: "SKYBAY"},
	}

	processor = []types.Processor{
		{Version: "pc-i440fx-7.0", ThreadCount: "1"},
		{Version: "Intel(R) Core(TM) i5-10210U CPU @ 1.60GHz", ThreadCount: "8"},
		{Version: "AMD Ryzen 3 3200G with Radeon Vega Graphics", ThreadCount: "4"},
		{Version: "Intel(R) Xeon(R) CPU E5-2620 0 @ 2.00GHz", ThreadCount: "12"},
	}

	memory = []types.Memory{
		{Manufacturer: "Kingston", Type: "DDR4"},
		{Manufacturer: "SK Hynix", Type: "DDR3"},
		{Manufacturer: "Hynix/Hyundai", Type: "DDR3"},
		{Manufacturer: "Hynix Semiconductor", Type: "DDR3"},
	}
)
