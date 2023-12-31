package crafter

import (
	"database/sql"
	"math/rand"
)

const (
	contractCreatedRatio = .1 // from devnet
	usedPublicIPsRatio   = .9
	nodeUpRatio          = .5
	maxContractHRU       = 1024 * 1024 * 1024 * 300
	maxContractSRU       = 1024 * 1024 * 1024 * 300
	maxContractMRU       = 1024 * 1024 * 1024 * 16
	maxContractCRU       = 16
	minContractHRU       = 0
	minContractSRU       = 1024 * 1024 * 256
	minContractMRU       = 1024 * 1024 * 256
	minContractCRU       = 1
)

var (
	r *rand.Rand

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
)

type Crafter struct {
	db *sql.DB

	nodesMRU               map[uint64]uint64
	nodesSRU               map[uint64]uint64
	nodesHRU               map[uint64]uint64
	nodeUP                 map[uint64]bool
	createdNodeContracts   []uint64
	dedicatedFarms         map[uint64]struct{}
	availableRentNodes     map[uint64]struct{}
	availableRentNodesList []uint64
	renter                 map[uint64]uint64

	NodeCount         uint
	FarmCount         uint
	PublicIPCount     uint
	TwinCount         uint
	NodeContractCount uint
	RentContractCount uint
	NameContractCount uint

	NodeStart     uint
	FarmStart     uint
	TwinStart     uint
	ContractStart uint
	BillStart     uint
	PublicIPStart uint
}

func NewCrafter(db *sql.DB,
	seed int,
	nodeCount,
	farmCount,
	twinCount,
	ipCount,
	nodeContractCount,
	nameContractCount,
	rentContractCount,
	nodeStart,
	farmStart,
	twinStart,
	contractStart,
	billStart,
	publicIPStart uint) Crafter {

	r = rand.New(rand.NewSource(int64(seed)))

	return Crafter{
		db: db,

		nodesMRU:               make(map[uint64]uint64),
		nodesSRU:               make(map[uint64]uint64),
		nodesHRU:               make(map[uint64]uint64),
		nodeUP:                 make(map[uint64]bool),
		createdNodeContracts:   make([]uint64, 0),
		dedicatedFarms:         make(map[uint64]struct{}),
		availableRentNodes:     make(map[uint64]struct{}),
		availableRentNodesList: make([]uint64, 0),
		renter:                 make(map[uint64]uint64),

		TwinCount:         twinCount,
		FarmCount:         farmCount,
		NodeCount:         nodeCount,
		PublicIPCount:     ipCount,
		NodeContractCount: nodeContractCount,
		RentContractCount: rentContractCount,
		NameContractCount: nameContractCount,

		NodeStart:     nodeStart,
		FarmStart:     farmStart,
		TwinStart:     twinStart,
		ContractStart: contractStart,
		BillStart:     billStart,
		PublicIPStart: publicIPStart,
	}
}

type contract_resources struct {
	id          string
	hru         uint64
	sru         uint64
	cru         uint64
	mru         uint64
	contract_id string
}
type farm struct {
	id                string
	grid_version      uint64
	farm_id           uint64
	name              string
	twin_id           uint64
	pricing_policy_id uint64
	certification     string
	stellar_address   string
	dedicated_farm    bool
}

type node struct {
	id                string
	grid_version      uint64
	node_id           uint64
	farm_id           uint64
	twin_id           uint64
	country           string
	city              string
	uptime            uint64
	created           uint64
	farming_policy_id uint64
	certification     string
	secure            bool
	virtualized       bool
	serial_number     string
	created_at        uint64
	updated_at        uint64
	location_id       string
	power             nodePower `gorm:"type:jsonb"`
	extra_fee         uint64
	dedicated         bool
}

type nodePower struct {
	State  string `json:"state"`
	Target string `json:"target"`
}
type twin struct {
	id           string
	grid_version uint64
	twin_id      uint64
	account_id   string
	relay        string
	public_key   string
}

type public_ip struct {
	id          string
	gateway     string
	ip          string
	contract_id uint64
	farm_id     string
}
type node_contract struct {
	id                    string
	grid_version          uint64
	contract_id           uint64
	twin_id               uint64
	node_id               uint64
	deployment_data       string
	deployment_hash       string
	number_of_public_i_ps uint64
	state                 string
	created_at            uint64
	resources_used_id     string
}
type node_resources_total struct {
	id      string
	hru     uint64
	sru     uint64
	cru     uint64
	mru     uint64
	node_id string
}
type public_config struct {
	id      string
	ipv4    string
	ipv6    string
	gw4     string
	gw6     string
	domain  string
	node_id string
}
type rent_contract struct {
	id           string
	grid_version uint64
	contract_id  uint64
	twin_id      uint64
	node_id      uint64
	state        string
	created_at   uint64
}
type location struct {
	id        string
	longitude string
	latitude  string
}

type contract_bill_report struct {
	id                string
	contract_id       uint64
	discount_received string
	amount_billed     uint64
	timestamp         uint64
}

type name_contract struct {
	id           string
	grid_version uint64
	contract_id  uint64
	twin_id      uint64
	name         string
	state        string
	created_at   uint64
}

type node_gpu struct {
	node_twin_id uint64
	id           string
	vendor       string
	device       string
	contract     int
}

type country struct {
	id         string
	country_id uint64
	code       string
	name       string
	region     string
	subregion  string
	lat        string
	long       string
}
