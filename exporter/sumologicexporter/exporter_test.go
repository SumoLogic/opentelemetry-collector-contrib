// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestInitExporter(t *testing.T) {
	_, err := initExporter(&Config{
		LogFormat:        "json",
		MetricFormat:     "carbon2",
		CompressEncoding: "gzip",
	})
	assert.NoError(t, err)
}

func TestInitExporterInvalidLogFormat(t *testing.T) {
	_, err := initExporter(&Config{
		LogFormat:        "test_format",
		MetricFormat:     "carbon2",
		CompressEncoding: "gzip",
	})

	assert.EqualError(t, err, "unexpected log format: test_format")
}

func TestInitExporterInvalidMetricFormat(t *testing.T) {
	_, err := initExporter(&Config{
		LogFormat:        "json",
		MetricFormat:     "test_format",
		CompressEncoding: "gzip",
	})

	assert.EqualError(t, err, "unexpected metric format: test_format")
}

func TestInitExporterInvalidCompressEncoding(t *testing.T) {
	_, err := initExporter(&Config{
		LogFormat:        "json",
		MetricFormat:     "carbon2",
		CompressEncoding: "test_format",
	})

	assert.EqualError(t, err, "unexpected compression encoding: test_format")
}

func TestAllLogsSuccess(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
			res.Write([]byte(""))
			body := extractBody(t, req)
			assert.Equal(t, body, `Example log`)
			assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "")
		},
	})
	defer func() { test.srv.Close() }()

	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	logs.ResourceLogs().At(0).InstrumentationLibraryLogs().Resize(1)
	for _, record := range exampleLog() {
		logs.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().Append(record)
	}

	_, err := test.exp.pushLogsData(context.Background(), logs)
	assert.NoError(t, err)
}

func TestAllLogsFailed(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(500)
			res.Write([]byte(""))
			body := extractBody(t, req)
			assert.Equal(t, body, "Example log\nAnother example log")
			assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "")
		},
	})
	defer func() { test.srv.Close() }()

	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	logs.ResourceLogs().At(0).InstrumentationLibraryLogs().Resize(1)
	for _, record := range exampleTwoLogs() {
		logs.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().Append(record)
	}

	dropped, err := test.exp.pushLogsData(context.Background(), logs)
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, dropped, 2)
}

func TestLogsPartiallyFailed(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(500)
			res.Write([]byte(""))
			body := extractBody(t, req)
			assert.Equal(t, body, "Example log")
			assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "key1=value1, key2=value2")
		},
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
			res.Write([]byte(""))
			body := extractBody(t, req)
			assert.Equal(t, body, "Another example log")
			assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "key3=value3, key4=value4")
		},
	})
	defer func() { test.srv.Close() }()

	f, err := newFilter([]string{`key\d`})
	require.NoError(t, err)
	test.exp.filter = f

	logs := pdata.NewLogs()
	logs.ResourceLogs().Resize(1)
	logs.ResourceLogs().At(0).InstrumentationLibraryLogs().Resize(1)
	for _, record := range exampleTwoDifferentLogs() {
		logs.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().Append(record)
	}

	dropped, err := test.exp.pushLogsData(context.Background(), logs)
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, dropped, 1)
}

func metricPairToMetrics(mp []metricPair) pdata.Metrics {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(len(mp))
	for num, record := range mp {
		metrics.ResourceMetrics().At(num).Resource().InitEmpty()
		record.attributes.CopyTo(metrics.ResourceMetrics().At(num).Resource().Attributes())
		metrics.ResourceMetrics().At(num).InstrumentationLibraryMetrics().Resize(1)
		metrics.ResourceMetrics().At(num).InstrumentationLibraryMetrics().At(0).Metrics().Append(record.metric)
	}

	return metrics
}

func TestAllMetricsSuccess(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, body, `test=test_value test2=second_value metric=test.metric.data unit=bytes  14500 1605534165
another_test=test_value metric=test.metric.data2 unit=s  123 1605534144
another_test=test_value metric=test.metric.data2 unit=s  124 1605534145`)
			assert.Equal(t, req.Header.Get("Content-Type"), "application/vnd.sumologic.carbon2")
		},
	})
	defer func() { test.srv.Close() }()
	metrics := metricPairToMetrics(exampleTwoIntMetrics())

	_, err := test.exp.pushMetricsData(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestAllMetricsFailed(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(500)
			body := extractBody(t, req)
			assert.Equal(t, body, `test=test_value test2=second_value metric=test.metric.data unit=bytes  14500 1605534165
another_test=test_value metric=test.metric.data2 unit=s  123 1605534144
another_test=test_value metric=test.metric.data2 unit=s  124 1605534145`)
			assert.Equal(t, req.Header.Get("Content-Type"), "application/vnd.sumologic.carbon2")
		},
	})
	defer func() { test.srv.Close() }()
	metrics := metricPairToMetrics(exampleTwoIntMetrics())

	dropped, err := test.exp.pushMetricsData(context.Background(), metrics)
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, dropped, 2)
}

func TestMetricsPartiallyFailed(t *testing.T) {
	test := prepareSenderTest(t, []func(res http.ResponseWriter, req *http.Request){
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(500)
			body := extractBody(t, req)
			assert.Equal(t, body, "test=test_value test2=second_value metric=test.metric.data unit=bytes  14500 1605534165")
			assert.Equal(t, req.Header.Get("Content-Type"), "application/vnd.sumologic.carbon2")
		},
		func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
			body := extractBody(t, req)
			assert.Equal(t, body, `another_test=test_value metric=test.metric.data2 unit=s  123 1605534144
another_test=test_value metric=test.metric.data2 unit=s  124 1605534145`)
			assert.Equal(t, req.Header.Get("Content-Type"), "application/vnd.sumologic.carbon2")
		},
	})
	defer func() { test.srv.Close() }()
	test.exp.config.MaxRequestBodySize = 1
	metrics := metricPairToMetrics(exampleTwoIntMetrics())

	dropped, err := test.exp.pushMetricsData(context.Background(), metrics)
	assert.EqualError(t, err, "error during sending data: 500 Internal Server Error")
	assert.Equal(t, dropped, 1)
}
