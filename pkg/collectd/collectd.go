package collectd

import (
	"log"

	"collectd.org/cdtime"
	jsoniter "github.com/json-iterator/go"
)

type collectdMeta struct {
	X map[string]interface{} `json:"-"` // Rest of the fields should go here.
}

// Collectd  ...
type Collectd struct {
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

// ParseInputString ...
func (c *Collectd) ParseInputString(jsonString string) (*[]Collectd, error) {
	jsonBlob := []byte(jsonString)

	return c.ParseInputByte(jsonBlob)
}

//ParseInputByte   ...
func (c *Collectd) ParseInputByte(jsonBlob []byte) (*[]Collectd, error) {
	collect := []Collectd{}
	//var json = jsoniter.ConfigCompatibleWithStandardLibrary.BorrowIterator(jsonBlob)
	var json = jsoniter.ConfigFastest.BorrowIterator(jsonBlob)
	//defer jsoniter.ConfigCompatibleWithStandardLibrary.ReturnIterator(json)
	defer jsoniter.ConfigFastest.ReturnIterator(json)
	json.ReadVal(&collect)
	//	err := json.Unmarshal(jsonBlob, &collect)
	if json.Error != nil {
		log.Println("Error parsing json:", json.Error)
		return nil, json.Error
	}

	return &collect, nil
}
