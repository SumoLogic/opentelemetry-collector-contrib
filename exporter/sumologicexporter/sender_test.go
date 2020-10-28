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

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type senderTest struct {
	srv *httptest.Server
	exp component.LogsExporter
	s   *sender
}

func prepareSenderTest(t *testing.T, cb []func(req *http.Request)) *senderTest {
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

	f, err := newFiltering([]string{})
	assert.NoError(t, err)

	return &senderTest{
		exp: exp,
		srv: testServer,
		s: newSender(
			cfg,
			&http.Client{
				Timeout: cfg.TimeoutSettings.Timeout,
			},
			f,
		),
	}
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

func TestSend(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {
		body := extractBody(t, req)
		assert.Equal(t, body, "Example log\nAnother example log")
		assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "test_metadata")
		assert.Equal(t, req.Header.Get("X-Sumo-Client"), "otelcol")
		assert.Equal(t, req.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}})
	defer func() { test.srv.Close() }()

	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].InitEmpty()
	buffer[1].Body().SetStringVal("Another example log")
	test.s.buffer = buffer

	err := test.s.sendLogs("test_metadata")
	assert.NoError(t, err)
}

func TestSendSplit(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){
		func(req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, body, "Example log")
		},
		func(req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, body, "Another example log")
		},
	})
	defer func() { test.srv.Close() }()
	test.s.config.MaxRequestBodySize = 10

	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].InitEmpty()
	buffer[1].Body().SetStringVal("Another example log")
	test.s.buffer = buffer

	err := test.s.sendLogs("test_metadata")
	assert.NoError(t, err)
}

func TestSendJson(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {
		body := extractBody(t, req)
		expected := `{"key1":"value1","key2":"value2","log":"Example log"}
{"key1":"value1","key2":"value2","log":"Another example log"}`
		assert.Equal(t, body, expected)
		assert.Equal(t, req.Header.Get("X-Sumo-Fields"), "test_metadata")
		assert.Equal(t, req.Header.Get("X-Sumo-Client"), "otelcol")
		assert.Equal(t, req.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}})
	defer func() { test.srv.Close() }()
	test.s.config.LogFormat = JSONFormat

	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")
	buffer[0].Attributes().InsertString("key1", "value1")
	buffer[0].Attributes().InsertString("key2", "value2")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].InitEmpty()
	buffer[1].Body().SetStringVal("Another example log")
	buffer[1].Attributes().InsertString("key1", "value1")
	buffer[1].Attributes().InsertString("key2", "value2")
	test.s.buffer = buffer

	err := test.s.sendLogs("test_metadata")
	assert.NoError(t, err)
}

func TestSendJsonSplit(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){
		func(req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, body, `{"key1":"value1","key2":"value2","log":"Example log"}`)
		},
		func(req *http.Request) {
			body := extractBody(t, req)
			assert.Equal(t, body, `{"key1":"value1","key2":"value2","log":"Another example log"}`)
		},
	})
	defer func() { test.srv.Close() }()
	test.s.config.LogFormat = JSONFormat
	test.s.config.MaxRequestBodySize = 10

	buffer := make([]pdata.LogRecord, 2)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")
	buffer[0].Attributes().InsertString("key1", "value1")
	buffer[0].Attributes().InsertString("key2", "value2")
	buffer[1] = pdata.NewLogRecord()
	buffer[1].InitEmpty()
	buffer[1].Body().SetStringVal("Another example log")
	buffer[1].Attributes().InsertString("key1", "value1")
	buffer[1].Attributes().InsertString("key2", "value2")
	test.s.buffer = buffer

	err := test.s.sendLogs("test_metadata")
	assert.Nil(t, err)
}

func TestSendUnexpectedFormat(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {}})
	defer func() { test.srv.Close() }()
	test.s.config.LogFormat = "dummy"

	buffer := make([]pdata.LogRecord, 1)
	buffer[0] = pdata.NewLogRecord()
	buffer[0].InitEmpty()
	buffer[0].Body().SetStringVal("Example log")

	err := test.s.sendLogs("test_metadata")
	assert.Error(t, err)
}

func TestOverrideSourceName(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {
		assert.Equal(t, req.Header.Get("X-Sumo-Name"), "Test source name")
	}})
	defer func() { test.srv.Close() }()

	test.s.config.SourceName = "Test source name"
	test.s.buffer = exampleLog()

	err := test.s.sendLogs("test_metadata")
	assert.NoError(t, err)
}

func TestOverrideSourceCategory(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {
		assert.Equal(t, req.Header.Get("X-Sumo-Category"), "Test source category")
	}})
	defer func() { test.srv.Close() }()

	test.s.config.SourceCategory = "Test source category"
	test.s.buffer = exampleLog()

	err := test.s.sendLogs("test_metadata")
	assert.Nil(t, err)
}

func TestOverrideSourceHost(t *testing.T) {
	test := prepareSenderTest(t, []func(req *http.Request){func(req *http.Request) {
		assert.Equal(t, req.Header.Get("X-Sumo-Host"), "Test source host")
	}})
	defer func() { test.srv.Close() }()

	test.s.config.SourceHost = "Test source host"
	test.s.buffer = exampleLog()

	err := test.s.sendLogs("test_metadata")
	assert.Nil(t, err)
}
