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
func (s *sender) send(pipeline string, body string, fields string) error {
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
		req.Header.Add("X-Sumo-Fields", fields)
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

// This function tries to send data and eventually appends error in case of failure
// It modifies buffer, droppedTimeSeries and errs
func (s *sender) sendAndPushErrors(previousMetadata string, droppedTimeSeries *int, errs *[]error) {

	err := s.sendLogs(previousMetadata)
	if err != nil {
		*droppedTimeSeries += len(s.buffer)
		*errs = append(*errs, err)
	}

	s.buffer = (s.buffer)[:0]
}

func (s *sender) sendLogs(fields string) error {
	switch s.config.LogFormat {
	case TextFormat:
		return s.sendLogsTextFormat(fields)
	case JSONFormat:
		return s.sendLogsJSONFormat(fields)
	default:
		return errors.New("Unexpected log format")
	}
}

func (s *sender) sendLogsTextFormat(fields string) error {
	body := strings.Builder{}
	var errs []error

	for j := 0; j < len(s.buffer); j++ {
		err := s.appendAndSend(s.buffer[j].Body().StringVal(), LogsPipeline, &body, fields)
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

func (s *sender) sendLogsJSONFormat(fields string) error {
	body := strings.Builder{}
	var errs []error

	for j := 0; j < len(s.buffer); j++ {
		data := s.filter.filterOut(s.buffer[j].Attributes())
		data[logKey] = s.buffer[j].Body().StringVal()

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
func (s *sender) appendAndSend(line string, pipeline string, body *strings.Builder, fields string) error {
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
