package sensu

import (
	"fmt"
	"strings"
)

type Message struct {
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
	StartsAt    string      `json:"startsAt"`
}

type Labels struct {
	Client   string `json:"client"`
	Check    string `json:"check"`
	Severity string `json:"severity"`
}

type Annotations struct {
	Command  string  `json:"command"`
	Issued   int64   `json:"issued"`
	Executed int64   `json:"executed"`
	Duration float64 `json:"duration"`
	Output   string  `json:"output"`
	Status   int     `json:"status"`
	StartsAt string  `json:"startsAt"`
}

type HealthCheckOutput []struct {
	Service   string  `json:"service"`
	Container string  `json:"container"`
	Status    string  `json:"status"`
	Healthy   float64 `json:"healthy"`
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
