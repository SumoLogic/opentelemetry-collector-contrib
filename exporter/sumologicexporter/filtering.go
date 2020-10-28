package sumologicexporter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type filtering struct {
	regexes []*regexp.Regexp
}

// FieldsType represents concatenated metadata
type FieldsType string

func newFiltering(fields []string) (*filtering, error) {
	metadataRegexes := make([]*regexp.Regexp, len(fields))

	for i, field := range fields {
		regex, err := regexp.Compile(field)
		if err != nil {
			return nil, err
		}

		metadataRegexes[i] = regex
	}

	return &filtering{
		regexes: metadataRegexes,
	}, nil
}

func (f *filtering) filter(attributes pdata.AttributeMap) map[string]string {
	returnValue := make(map[string]string)

	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		for _, regex := range f.regexes {
			if regex.MatchString(k) {
				returnValue[k] = v.StringVal()
				return
			}
		}
	})
	return returnValue
}

func (f *filtering) filterOut(attributes pdata.AttributeMap) map[string]string {
	returnValue := make(map[string]string)

	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		for _, regex := range f.regexes {
			if regex.MatchString(k) {
				return
			}
		}
		returnValue[k] = v.StringVal()
	})
	return returnValue
}

// GetMetadata builds string which represents metadata in alphabetical order
func (f *filtering) GetMetadata(attributes pdata.AttributeMap) FieldsType {
	attrs := f.filter(attributes)
	metadata := make([]string, 0, len(attrs))

	for k, v := range attrs {
		metadata = append(metadata, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(metadata)

	return FieldsType(strings.Join(metadata, ", "))
}
