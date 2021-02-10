package lib

import "fmt"

func ceilometerEventFormatter(record map[string]interface{}) error {
	// transforms traits key into map[string]interface{}
	if payload, ok := record["payload"]; ok {
		newPayload := make(map[string]interface{})
		if typedPayload, ok := payload.(map[string]interface{}); ok {
			if traitData, ok := typedPayload["traits"]; ok {
				if traits, ok := traitData.([]interface{}); ok {
					newTraits := make(map[string]interface{})
					for _, value := range traits {
						if typedValue, ok := value.([]interface{}); ok {
							if len(typedValue) != 3 {
								return fmt.Errorf("parsed invalid trait in event: '%v'", value)
							}
							if traitType, ok := typedValue[1].(float64); ok {
								switch traitType {
								case 2:
									newTraits[typedValue[0].(string)] = typedValue[2].(float64)
								default:
									newTraits[typedValue[0].(string)] = typedValue[2].(string)
								}
							} else {
								return fmt.Errorf("parsed invalid trait in event: '%v'", value)
							}
						} else {
							return fmt.Errorf("parsed invalid trait in event: '%v'", value)
						}
					}
					newPayload["traits"] = newTraits
				}
			}
			for key, value := range typedPayload {
				if key != "traits" {
					newPayload[key] = value
				}
			}
		}
		record["payload"] = newPayload
	}
	return nil
}

func collectdEventFormatter(record map[string]interface{}) error {
	//noop since collectd events does not need any special formatting (yet?)
	return nil
}

//EventFormatters are special functions used to reorganize event structures in case of need.
//Each data source has one formatter which should handle message formatting completely
var EventFormatters = map[string]func(map[string]interface{}) error{
	"ceilometer": ceilometerEventFormatter,
	"collectd":   collectdEventFormatter,
}
