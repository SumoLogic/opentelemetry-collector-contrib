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
	var expected FieldsType = "key1=value1, key2=value2, key3=value3"
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

func TestConvertStringAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("key", "test_value")

	regexes := []string{}
	f, err := newFiltering(regexes)
	assert.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "test_value")
}

func TestConvertIntAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertInt("key", 15)

	regexes := []string{}
	f, err := newFiltering(regexes)
	assert.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "15")
}

func TestConvertDoubleAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertDouble("key", 4.16)

	regexes := []string{}
	f, err := newFiltering(regexes)
	assert.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "4.16")
}

func TestConvertBoolAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertBool("key", false)

	regexes := []string{}
	f, err := newFiltering(regexes)
	require.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "false")
}
