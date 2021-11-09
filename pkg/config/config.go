package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

var (
	// Validate holds config validator
	Validate = validator.New()
)

// ParseConfig parses and validates input into config object
func ParseConfig(r io.Reader, config interface{}) error {
	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "while reading configuration")
	}
	if string(configBytes) == "null\n" {
		return nil
	}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		return errors.Wrap(err, "unmarshalling config yaml")
	}

	err = Validate.Struct(config)
	if err != nil {
		if e, ok := err.(validator.ValidationErrors); ok {
			missingFields := []string{}
			for _, fe := range e {
				missingFields = append(missingFields, setCamelCase(fe.Namespace()))
			}
			return fmt.Errorf("missing or incorrect configuration fields --  %s --", strings.Join(missingFields, " , "))
		}
		return errors.Wrap(err, "error while validating configuration")
	}
	return nil
}

func setCamelCase(field string) string {
	items := strings.Split(field, ".")
	ret := []string{}
	for _, item := range items {
		camel := []byte(item)
		l := bytes.ToLower([]byte{camel[0]})
		camel[0] = l[0]
		ret = append(ret, string(camel))
	}
	return strings.Join(ret[1:], ".")
}
