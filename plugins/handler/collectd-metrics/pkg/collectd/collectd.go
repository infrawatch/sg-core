package collectd

import (
	"collectd.org/cdtime"
	jsoniter "github.com/json-iterator/go"
)

type collectdMeta struct {
	X map[string]interface{} `json:"-"` // Rest of the fields should go here.
}

// Metric  ...
type Metric struct {
	Values         []float64    `json:"values"`
	Dstypes        []string     `json:"dstypes"`
	Dsnames        []string     `json:"dsnames,omitempty"`
	Time           cdtime.Time  `json:"time"`
	Interval       float64      `json:"interval"`
	Host           string       `json:"host"`
	Plugin         string       `json:"plugin"`
	PluginInstance string       `json:"plugin_instance,omitempty"`
	Type           string       `json:"type"`
	TypeInstance   string       `json:"type_instance,omitempty"`
	Meta           collectdMeta `json:"meta,omitempty"`
}

//ParseInputByte   ...
func ParseInputByte(jsonBlob []byte) (*[]Metric, error) {
	collect := []Metric{}
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary.BorrowIterator(jsonBlob)
	var json = jsoniter.ConfigFastest.BorrowIterator(jsonBlob)
	//defer jsoniter.ConfigCompatibleWithStandardLibrary.ReturnIterator(json)
	json.ReadVal(&collect)
	//	err := json.Unmarshal(jsonBlob, &collect)
	if json.Error != nil {
		return nil, json.Error
	}
	jsoniter.ConfigFastest.ReturnIterator(json)
	return &collect, nil
}
