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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

type test struct {
	srv *httptest.Server
	exp component.LogsExporter
	se  *sumologicexporter
	s   *sender
}

func getExporter(t *testing.T, cb []func(req *http.Request)) *test {
	reqCounter := 0
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte(""))
		if len(cb) > 0 && assert.Greater(t, len(cb), reqCounter) {
			cb[reqCounter](req)
			reqCounter++
		}
	}))

	cfg := &Config{
		URL:                testServer.URL,
		LogFormat:          "text",
		Client:             "otelcol",
		MaxRequestBodySize: 20_971_520,
		TimeoutSettings:    CreateDefaultTimeoutSettings(),
	}
	factory := NewFactory()
	exp, err := factory.CreateLogsExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)

	assert.NoError(t, err)

	se, err := initExporter(cfg)
	assert.NoError(t, err)

	f, err := newFiltering([]string{})
	assert.NoError(t, err)

	return &test{
		exp: exp,
		srv: testServer,
		se:  se,
		s:   newSender(se.config, se.client, f),
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

	test := getExporter(t, []func(req *http.Request){func(req *http.Request) {}})
	test.se.config.MetadataFields = []string{"^key[12]", "^key3"}
	test.se.refreshMetadataRegexes()

	data := test.se.filterMetadata(attributes, true)
	expected := map[string]string{
		"additional_key2": "value2",
		"additional_key3": "value3",
	}
	assert.Equal(t, data, expected)
}

func extractBody(t *testing.T, req *http.Request) string {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, req.Body)
	assert.NoError(t, err)
	return buf.String()
}

func exampleLog() []pdata.LogRecord {
	buffer := make([]pdata.LogRecord, 1)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")

	return buffer
}
