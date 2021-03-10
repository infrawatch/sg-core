package lib

import (
	"fmt"
	"log"
	"strings"
)

// assimilateMap recursively saves content of the given map to destination map of strings
func assimilateMap(theMap map[string]interface{}, destination *map[string]string) {
	defer func() { // recover from any panic
		if r := recover(); r != nil {
			log.Printf("Panic:recovered in assimilateMap %v\n", r)
		}
	}()
	for key, val := range theMap {
		switch value := val.(type) {
		case map[string]interface{}:
			// go one level deeper in the map
			assimilateMap(value, destination)
		case []interface{}:
			// transform slice value to comma separated list and assimilate it
			aList := make([]string, 0, len(value))
			for _, item := range value {
				if itm, ok := item.(string); ok {
					aList = append(aList, itm)
				}
			}
			(*destination)[key] = strings.Join(aList, ",")
		case float64, float32:
			(*destination)[key] = fmt.Sprintf("%f", value)
		case int:
			(*destination)[key] = fmt.Sprintf("%d", value)
		default:
			// assimilate KV pair
			(*destination)[key] = value.(string)
		}
	}
}
