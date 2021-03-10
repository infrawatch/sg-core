package lib

import (
	"regexp"
)

var (
	// Ceilometer data parsers
	rexForOsloMessage = regexp.MustCompile(`\\*"oslo.message\\*"\s*:\s*\\*"({.*})\\*"`)
	rexForPayload     = regexp.MustCompile(`\\+"payload\\+"\s*:\s*\[(.*)\]`)
	rexForEventType   = regexp.MustCompile(`\\+"event_type\\+"\s*:\s*\\*"`)
	// collectd data parsers
	rexForLabelsField     = regexp.MustCompile(`\\?"labels\\?"\w?:\w?\{`)
	rexForAnnotationField = regexp.MustCompile(`\\?"annotations\\?"\w?:\w?\{`)
)

func recognizeCeilometer(jsondata []byte) bool {
	match := rexForOsloMessage.FindSubmatchIndex(jsondata)
	if match == nil {
		return false
	}
	match = rexForEventType.FindSubmatchIndex(jsondata)
	if match == nil {
		return false
	}
	match = rexForPayload.FindSubmatchIndex(jsondata)
	return match != nil
}

func recognizeCollectd(jsondata []byte) bool {
	labels := rexForLabelsField.FindIndex(jsondata)
	annots := rexForAnnotationField.FindIndex(jsondata)
	return labels != nil && annots != nil
}

var recognizers = map[string](func([]byte) bool){
	"collectd":   recognizeCollectd,
	"ceilometer": recognizeCeilometer,
}

// DataSource indentifies a format of incoming data in the message bus channel.
type DataSource int

// ListAll returns slice of supported data sources in form of human readable names.
func (src DataSource) ListAll() []string {
	return []string{"ceilometer", "collectd", "generic"}
}

// SetFromString resets value according to given human readable identification. Returns false if invalid identification was given.
func (src *DataSource) SetFromString(name string) bool {
	for index, value := range src.ListAll() {
		if name == value {
			*src = DataSource(index)
			return true
		}
	}
	return false
}

// SetFromMessage resets value according to given message data format
func (src *DataSource) SetFromMessage(jsondata []byte) {
	for source, rec := range recognizers {
		if rec(jsondata) {
			src.SetFromString(source)
			return
		}
	}
	//TODO: right now generic event message is everything else than collectd or ceilometer event,
	//      but once we come up with SG generic event format, we need to add it's recognizer
	src.SetFromString("generic")
}

// String returns human readable data type identification.
func (src DataSource) String() string {
	return (src.ListAll())[src]
}
