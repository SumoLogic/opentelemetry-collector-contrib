// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func exampleTwoIntMetrics() []metricPair {
	dataPoint := pdata.NewIntDataPoint()
	dataPoint.InitEmpty()
	dataPoint.SetTimestamp(1605534165 * 1e9)
	dataPoint.SetValue(14500)

	dataPoint2 := pdata.NewIntDataPoint()
	dataPoint2.InitEmpty()
	dataPoint2.SetTimestamp(1605534144 * 1e9)
	dataPoint2.SetValue(123)

	dataPoint3 := pdata.NewIntDataPoint()
	dataPoint3.InitEmpty()
	dataPoint3.SetTimestamp(1605534145 * 1e9)
	dataPoint3.SetValue(124)

	metric := pdata.NewMetric()
	metric.InitEmpty()
	metric.SetName("test.metric.data")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeIntSum)
	metric.IntSum().InitEmpty()
	metric.IntSum().DataPoints().Append(dataPoint)

	metric2 := pdata.NewMetric()
	metric2.InitEmpty()
	metric2.SetName("test.metric.data2")
	metric2.SetUnit("s")
	metric2.SetDataType(pdata.MetricDataTypeIntGauge)
	metric2.IntGauge().InitEmpty()
	metric2.IntGauge().DataPoints().Append(dataPoint2)
	metric2.IntGauge().DataPoints().Append(dataPoint3)

	attributes := pdata.NewAttributeMap()
	attributes.InsertString("test", "test_value")
	attributes.InsertString("test2", "second_value")

	attributes2 := pdata.NewAttributeMap()
	attributes2.InsertString("another_test", "test_value")

	return []metricPair{
		{
			metric:     metric,
			attributes: attributes,
		},
		{
			metric:     metric2,
			attributes: attributes2,
		},
	}
}

func exampleTwoDoubleMetrics() []metricPair {
	dataPoint := pdata.NewDoubleDataPoint()
	dataPoint.InitEmpty()
	dataPoint.SetTimestamp(1605534165 * 1e9)
	dataPoint.SetValue(14.500)

	dataPoint2 := pdata.NewDoubleDataPoint()
	dataPoint2.InitEmpty()
	dataPoint2.SetTimestamp(1605534144 * 1e9)
	dataPoint2.SetValue(1.23)

	dataPoint3 := pdata.NewDoubleDataPoint()
	dataPoint3.InitEmpty()
	dataPoint3.SetTimestamp(1605534145 * 1e9)
	dataPoint3.SetValue(1.24)

	metric := pdata.NewMetric()
	metric.InitEmpty()
	metric.SetName("test.metric.data")
	metric.SetUnit("bytes")
	metric.SetDataType(pdata.MetricDataTypeDoubleSum)
	metric.DoubleSum().InitEmpty()
	metric.DoubleSum().DataPoints().Append(dataPoint)

	metric2 := pdata.NewMetric()
	metric2.InitEmpty()
	metric2.SetName("test.metric.data2")
	metric2.SetUnit("s")
	metric2.SetDataType(pdata.MetricDataTypeDoubleGauge)
	metric2.DoubleGauge().InitEmpty()
	metric2.DoubleGauge().DataPoints().Append(dataPoint2)
	metric2.DoubleGauge().DataPoints().Append(dataPoint3)

	attributes := pdata.NewAttributeMap()
	attributes.InsertString("test", "test_value")
	attributes.InsertString("test2", "second_value")

	attributes2 := pdata.NewAttributeMap()
	attributes2.InsertString("another_test", "test_value")

	return []metricPair{
		{
			metric:     metric,
			attributes: attributes,
		},
		{
			metric:     metric2,
			attributes: attributes2,
		},
	}
}

func TestGetMetadata(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("key3", "value3")
	attributes.InsertString("key1", "value1")
	attributes.InsertString("key2", "value2")
	attributes.InsertString("additional_key2", "value2")
	attributes.InsertString("additional_key3", "value3")

	regexes := []string{"^key[12]", "^key3"}
	f, err := newFilter(regexes)
	require.NoError(t, err)

	metadata := f.GetMetadata(attributes)
	const expected Fields = "key1=value1, key2=value2, key3=value3"
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
	f, err := newFilter(regexes)
	require.NoError(t, err)

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
	f, err := newFilter(regexes)
	require.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "test_value")
}

func TestConvertIntAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertInt("key", 15)

	regexes := []string{}
	f, err := newFilter(regexes)
	require.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "15")
}

func TestConvertDoubleAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertDouble("key", 4.16)

	regexes := []string{}
	f, err := newFilter(regexes)
	require.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "4.16")
}

func TestConvertBoolAttributeToString(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertBool("key", false)

	regexes := []string{}
	f, err := newFilter(regexes)
	require.NoError(t, err)

	value, _ := attributes.Get("key")
	data := f.convertAttributeToString(value)
	assert.Equal(t, data, "false")
}

func TestCarbon2TagString(t *testing.T) {
	regexes := []string{}
	f, err := newFilter(regexes)
	require.NoError(t, err)

	metric := exampleTwoIntMetrics()[0]
	data := f.Carbon2TagString(metric)
	assert.Equal(t, "test=test_value test2=second_value metric=test.metric.data unit=bytes", data)

	metric = exampleTwoIntMetrics()[1]
	data = f.Carbon2TagString(metric)
	assert.Equal(t, "another_test=test_value metric=test.metric.data2 unit=s", data)

	metric = exampleTwoDoubleMetrics()[0]
	data = f.Carbon2TagString(metric)
	assert.Equal(t, "test=test_value test2=second_value metric=test.metric.data unit=bytes", data)

	metric = exampleTwoDoubleMetrics()[1]
	data = f.Carbon2TagString(metric)
	assert.Equal(t, "another_test=test_value metric=test.metric.data2 unit=s", data)
}
