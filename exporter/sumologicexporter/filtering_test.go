package sumologicexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestGetMetadata(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("key3", "value3")
	attributes.InsertString("key1", "value1")
	attributes.InsertString("key2", "value2")
	attributes.InsertString("additional_key2", "value2")
	attributes.InsertString("additional_key3", "value3")

	regexes := []string{"^key[12]", "^key3"}
	f, err := newFiltering(regexes)
	assert.NoError(t, err)

	metadata := f.GetMetadata(attributes)
	expected := "key1=value1, key2=value2, key3=value3"
	assert.Equal(t, expected, metadata)
}

func TestFilterOutMetadata(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("key3", "value3")
	attributes.InsertString("key1", "value1")
	attributes.InsertString("key2", "value2")
	attributes.InsertString("additional_key2", "value2")
	attributes.InsertString("additional_key3", "value3")

	regexes := []string{"^key[12]", "^key3"}
	f, err := newFiltering(regexes)
	assert.NoError(t, err)

	data := f.filterOut(attributes)
	expected := map[string]string{
		"additional_key2": "value2",
		"additional_key3": "value3",
	}
	assert.Equal(t, data, expected)
}
