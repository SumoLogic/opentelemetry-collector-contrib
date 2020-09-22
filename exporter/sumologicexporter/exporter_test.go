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
	se  *sumologicexporter
}

func getExporter(t *testing.T, cb func(req *http.Request)) *test {
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(200)
		res.Write([]byte(""))
		if cb != nil {
			cb(req)
		}
	}))
	// defer func() { testServer.Close() }()

	cfg := &Config{
		Endpoint: testServer.URL,
	}
	factory := NewFactory()
	se, err := factory.CreateLogsExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)

	assert.NoError(t, err)

	return &test{
		se:  se.(*sumologicexporter),
		srv: testServer,
	}
}

func TestGetMetadata(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("key3", "value3")
	attributes.InsertString("key1", "value1")
	attributes.InsertString("key2", "value2")

	test := getExporter(t, func(req *http.Request) {})
	metadata := test.se.GetMetadata(attributes)
	expected := "key1=value1, key2=value2, key3=value3"
	assert.Equal(t, expected, metadata)
}

func extractBody(req *http.Request) string {
	buf := new(strings.Builder)
	io.Copy(buf, req.Body)
	return buf.String()
}

func TestSend(t *testing.T) {
	test := getExporter(t, func(req *http.Request) {
		body := extractBody(req)
		assert.Equal(t, body, "Example log\nAnother example log")
		assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "")
		assert.Equal(t, req.Header.Get("X-Sumo-Name"), "otelcol")
		assert.Equal(t, req.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	})
	defer func() { test.srv.Close() }()

	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].InitEmpty()
	buffer[1].Body().SetStringVal("Another example log")
	test.se.Send(buffer, test.se.GetMetadata(buffer[0].Attributes()))
}
