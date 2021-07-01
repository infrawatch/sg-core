package sensu

import (
	"fmt"
	"strings"
)

type Message struct {
	Labels      Labels
	Annotations Annotations
	StartsAt    string
}

type Labels struct {
	Client   string
	Check    string
	Severity string
}

type Annotations struct {
	Command  string
	Issued   int64
	Executed int64
	Duration float64
	Output   string
	Status   int
	Ves      string
	StartsAt string
}

type HealthCheckOutput []struct {
	Service   string
	Container int64
	Status    string
	Healthy   float64
}

type ErrMissingFields struct {
	Fields []string
}

func (eMF *ErrMissingFields) Error() string {
	fieldsFormatted := strings.Join(eMF.Fields, ", ")
	return fmt.Sprintf("missing fields in received data (%s)", fieldsFormatted)
}

func (eMF *ErrMissingFields) addMissingField(f string) {
	eMF.Fields = append(eMF.Fields, f)
}

func IsMsgValid(msg Message) bool {
	if msg.StartsAt == "" {
		return false
	}

	if msg.Labels.Client == "" {
		return false
	}
	return true
}

func IsOutputValid(outputs HealthCheckOutput) bool {
	for _, out := range outputs {
		if out.Service == "" {
			return false
		}
	}
	return true
}

func BuildMsgErr(msg Message) error {
	err := &ErrMissingFields{}
	if msg.StartsAt == "" {
		err.addMissingField("startsAt")
	}

	if msg.Labels.Client == "" {
		err.addMissingField("labels.client")
	}
	return err
}

func BuildOutputsErr(outputs HealthCheckOutput) error {
	err := &ErrMissingFields{}
	for index, out := range outputs {
		if out.Service == "" {
			err.addMissingField(fmt.Sprintf("annotations.output[%d].service", index))
		}
	}
	return err
}
