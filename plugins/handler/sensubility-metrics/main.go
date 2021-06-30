package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/infrawatch/sg-core/pkg/bus"
	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type ErrMissingFields struct {
	fields []string
}

func (eMF *ErrMissingFields) Error() string {
	fieldsFormatted := strings.Join(eMF.fields, ", ")
	return fmt.Sprintf("missing fields in received data (%s)", fieldsFormatted)
}

func (eMF *ErrMissingFields) addMissingField(f string) {
	eMF.fields = append(eMF.fields, f)
}

type sensubilityMetrics struct{}

func (sm *sensubilityMetrics) Run(ctx context.Context, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) {

}

func (sm *sensubilityMetrics) Handle(blob []byte, reportErrors bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	data := new(map[string]interface{})
	err := json.Unmarshal(blob, &data)
	if err != nil {
		return err
	}

	fmt.Println(string(blob))

	return nil
}

func (sm *sensubilityMetrics) Identify() string {
	return "sensubility-metrics"
}

func (sm *sensubilityMetrics) Config(blob []byte) error {
	return nil
}
