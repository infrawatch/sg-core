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

//ParseConfig parses and validates input into config object
func ParseConfig(r io.Reader, config interface{}) error {
	validate := validator.New()
	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "while reading configuration")
	}
	if "null\n" == string(configBytes) {
		return nil
	}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		return errors.Wrap(err, "unmarshalling config yaml")
	}

	err = validate.Struct(config)
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
