// Copyright 2019, OpenTelemetry Authors
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
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type sender struct {
	buffer []pdata.LogRecord
	config *Config
	client *http.Client
	filter *filtering
}

func newSender(cfg *Config, cl *http.Client, f *filtering) *sender {
	return &sender{
		config: cfg,
		client: cl,
		filter: f,
	}
}

// Send sends data to sumologic
func (s *sender) send(pipeline PipelineType, body string, fields FieldsType) error {
	// Add headers
	req, err := http.NewRequest(http.MethodPost, s.config.URL, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Add("X-Sumo-Client", s.config.Client)

	if len(s.config.SourceHost) > 0 {
		req.Header.Add("X-Sumo-Host", s.config.SourceHost)
	}

	if len(s.config.SourceName) > 0 {
		req.Header.Add("X-Sumo-Name", s.config.SourceName)
	}

	if len(s.config.SourceCategory) > 0 {
		req.Header.Add("X-Sumo-Category", s.config.SourceCategory)
	}

	switch pipeline {
	case LogsPipeline:
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Sumo-Fields", string(fields))
	case MetricsPipeline:
		// ToDo: Implement metrics pipeline
		return errors.New("Current sender version doesn't support metrics")
	default:
		return errors.New("Unexpected pipeline")
	}

	_, err = s.client.Do(req)
	// ToDo: Add retries mechanism
	if err != nil {
		return err
	}
	return nil
}

func (s *sender) sendLogs(fields FieldsType) error {
	switch s.config.LogFormat {
	case TextFormat:
		return s.sendLogsTextFormat(fields)
	case JSONFormat:
		return s.sendLogsJSONFormat(fields)
	default:
		return errors.New("Unexpected log format")
	}
}

func (s *sender) sendLogsTextFormat(fields FieldsType) error {
	var (
		body strings.Builder
		errs []error
	)

	for _, record := range s.buffer {
		err := s.appendAndSend(record.Body().StringVal(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := s.send(LogsPipeline, body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

func (s *sender) sendLogsJSONFormat(fields FieldsType) error {
	var (
		body strings.Builder
		errs []error
	)

	for _, record := range s.buffer {
		data := s.filter.filterOut(record.Attributes())
		data[logKey] = record.Body().StringVal()

		nextLine, err := json.Marshal(data)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = s.appendAndSend(bytes.NewBuffer(nextLine).String(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := s.send(LogsPipeline, body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

// appendAndSend appends line to the body and eventually sends data to avoid exceeding the request limit
func (s *sender) appendAndSend(line string, pipeline PipelineType, body *strings.Builder, fields FieldsType) error {
	var err error

	if body.Len() > 0 && body.Len()+len(line) > s.config.MaxRequestBodySize {
		err = s.send(LogsPipeline, body.String(), fields)
		body.Reset()
	}

	if body.Len() > 0 {
		// Do not add newline if the body is empty
		body.WriteString("\n")
	}

	body.WriteString(line)
	return err
}

// clean buffer zeroes buffer
func (s *sender) cleanBuffer() {
	s.buffer = (s.buffer)[:0]
}

// append adds log to the buffer
func (s *sender) appendLog(log pdata.LogRecord) {
	s.buffer = append(s.buffer, log)
}

// count returns number of logs in buffer
func (s *sender) count() int {
	return len(s.buffer)
}
