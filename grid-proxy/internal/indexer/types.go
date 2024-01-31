package indexer

// DMI represents a map of SectionTypeStr to Section parsed from dmidecode output,
// as well as information about the tool used to get these sections
// Property in section is in the form of key value pairs where values are optional
// and may include a list of items as well.
// k: [v]
//
//	[
//		item1
//		item2
//		...
//	]
type DMI struct {
	Tooling  Tooling   `json:"tooling"`
	Sections []Section `json:"sections"`
}

// Tooling holds the information and version about the tool used to
// read DMI information
type Tooling struct {
	Aggregator string `json:"aggregator"`
	Decoder    string `json:"decoder"`
}

// Section represents a complete section like BIOS or Baseboard
type Section struct {
	HandleLine  string       `json:"handleline"`
	TypeStr     string       `json:"typestr,omitempty"`
	Type        Type         `json:"typenum"`
	SubSections []SubSection `json:"subsections"`
}

// Type (allowed types 0 -> 42)
type Type int

// SubSection represents part of a section, identified by a title
type SubSection struct {
	Title      string                  `json:"title"`
	Properties map[string]PropertyData `json:"properties,omitempty"`
}

// PropertyData represents a key value pair with optional list of items
type PropertyData struct {
	Val   string   `json:"value"`
	Items []string `json:"items,omitempty"`
}
