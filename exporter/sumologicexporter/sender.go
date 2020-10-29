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
	"fmt"
	"io"
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
func (s *sender) send(pipeline PipelineType, body io.Reader, fields FieldsType) error {
	// Add headers
	req, err := http.NewRequest(http.MethodPost, s.config.URL, body)
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

	resp, err := s.client.Do(req)
	// ToDo: Add retries mechanism
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("Error during sending data: %s", resp.Status)
	}
	return nil
}

func (s *sender) sendLogs(fields FieldsType) (int, error) {
	switch s.config.LogFormat {
	case TextFormat:
		return s.sendLogsTextFormat(fields)
	case JSONFormat:
		return s.sendLogsJSONFormat(fields)
	default:
		return s.count(), errors.New("Unexpected log format")
	}
}

func (s *sender) sendLogsTextFormat(fields FieldsType) (int, error) {
	var (
		body              strings.Builder
		errs              []error
		droppedTimeSeries int
		currentTimeSeries int
	)

	for _, record := range s.buffer {
		sent, appended, err := s.appendAndSend(record.Body().StringVal(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
			if sent {
				droppedTimeSeries += currentTimeSeries
			}

			if !appended {
				droppedTimeSeries++
			}
		}

		// If data was sent, cleanup the currentTimeSeries counter
		if sent {
			currentTimeSeries = 0
		}

		// If log has been appended to body, increment the currentTimeSeries
		if appended {
			currentTimeSeries++
		}
	}

	err := s.send(LogsPipeline, strings.NewReader(body.String()), fields)
	if err != nil {
		errs = append(errs, err)
		droppedTimeSeries += currentTimeSeries
	}

	if len(errs) > 0 {
		return droppedTimeSeries, componenterror.CombineErrors(errs)
	}
	return droppedTimeSeries, nil
}

func (s *sender) sendLogsJSONFormat(fields FieldsType) (int, error) {
	var (
		body              strings.Builder
		errs              []error
		droppedTimeSeries int
		currentTimeSeries int
	)

	for _, record := range s.buffer {
		data := s.filter.filterOut(record.Attributes())
		data[logKey] = record.Body().StringVal()

		nextLine, err := json.Marshal(data)
		if err != nil {
			droppedTimeSeries++
			errs = append(errs, err)
			continue
		}

		sent, appended, err := s.appendAndSend(bytes.NewBuffer(nextLine).String(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
			if sent {
				droppedTimeSeries += currentTimeSeries
			}

			if !appended {
				droppedTimeSeries++
			}
		}

		// If data was sent, cleanup the currentTimeSeries counter
		if sent {
			currentTimeSeries = 0
		}

		// If log has been appended to body, increment the currentTimeSeries
		if appended {
			currentTimeSeries++
		}
	}

	err := s.send(LogsPipeline, strings.NewReader(body.String()), fields)
	if err != nil {
		errs = append(errs, err)
		droppedTimeSeries += currentTimeSeries
	}

	if len(errs) > 0 {
		return droppedTimeSeries, componenterror.CombineErrors(errs)
	}
	return droppedTimeSeries, nil
}

// appendAndSend appends line to the body and eventually sends data to avoid exceeding the request limit
func (s *sender) appendAndSend(line string, pipeline PipelineType, body *strings.Builder, fields FieldsType) (bool, bool, error) {
	var errors []error
	// sent gives information if the data was sent or not
	sent := false
	// appended keeps state of appending new log line to the body
	appended := true

	if body.Len() > 0 && body.Len()+len(line) > s.config.MaxRequestBodySize {
		err := s.send(LogsPipeline, strings.NewReader(body.String()), fields)
		sent = true
		if err != nil {
			errors = append(errors, err)
		}
		body.Reset()
	}

	if body.Len() > 0 {
		// Do not add newline if the body is empty
		_, err := body.WriteString("\n")
		if err != nil {
			errors = append(errors, err)
			appended = false
		}
	}

	_, err := body.WriteString(line)
	if err != nil && appended {
		errors = append(errors, err)
		appended = false
	}

	if len(errors) > 0 {
		return sent, appended, componenterror.CombineErrors(errors)
	}
	return sent, appended, nil
}

// cleanBuffer zeroes buffer
func (s *sender) cleanBuffer() {
	s.buffer = (s.buffer)[:0]
}

// append adds log to the buffer
func (s *sender) batch(log pdata.LogRecord, metadata FieldsType) (int, error) {
	s.buffer = append(s.buffer, log)

	if s.count() == maxBufferSize {
		dropped, err := s.sendLogs(metadata)
		s.cleanBuffer()
		return dropped, err
	}

	return 0, nil
}

// count returns number of logs in buffer
func (s *sender) count() int {
	return len(s.buffer)
}
