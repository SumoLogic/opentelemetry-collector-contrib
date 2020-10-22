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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type sumologicexporter struct {
	config          *Config
	metadataRegexes []*regexp.Regexp
}

func newLogsExporter(
	cfg *Config,
) (component.LogsExporter, error) {
	se := &sumologicexporter{
		config: cfg,
	}
	err := se.refreshMetadataRegexes()

	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogsExporter(
		cfg,
		se.pushLogsData,
		// Disable exporterhelper Timeout, since we are using a custom mechanism
		// within exporter itself
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)
}

func (se *sumologicexporter) refreshMetadataRegexes() error {
	cfg := se.config
	metadataRegexes := make([]*regexp.Regexp, len(cfg.MetadataFields))
	for i := 0; i < len(cfg.MetadataFields); i++ {
		regex, err := regexp.Compile(cfg.MetadataFields[i])
		if err != nil {
			return err
		}
		metadataRegexes[i] = regex
	}

	se.metadataRegexes = metadataRegexes
	return nil
}

// filterMetadata returns map of attributes which are (or are not, it depends on filterOut argument) metadata
func (se *sumologicexporter) filterMetadata(attributes pdata.AttributeMap, filterOut bool) map[string]string {
	returnValue := make(map[string]string)
	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		skip := !filterOut
		for j := 0; j < len(se.metadataRegexes); j++ {
			if se.metadataRegexes[j].MatchString(k) {
				skip = false
				if filterOut {
					return
				}
			}
		}

		if skip {
			return
		}

		returnValue[k] = v.StringVal()
	})

	return returnValue
}

// GetMetadata builds string which represents metadata in alphabetical order
func (se *sumologicexporter) GetMetadata(attributes pdata.AttributeMap) string {
	attrs := se.filterMetadata(attributes, false)
	keys := make([]string, 0, len(attrs))
	buf := strings.Builder{}

	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s=%s", k, attrs[k]))
	}

	return buf.String()
}

// This function tries to send data and modify pass values
func (se *sumologicexporter) sendAndPushErrors(buffer *[]pdata.LogRecord, fields string, droppedTimeSeries *int, errors *[]error) {
	err := se.sendLogs(*buffer, fields)
	if err != nil {
		*droppedTimeSeries += len(*buffer)
		*errors = append(*errors, err)
	}
	*buffer = (*buffer)[:0]
}

// pushLogsData groups data with common metadata uses Send to send data to sumologic
func (se *sumologicexporter) pushLogsData(ctx context.Context, ld pdata.Logs) (droppedTimeSeries int, err error) {
	maxBufferSize := 100
	buffer := make([]pdata.LogRecord, 0, maxBufferSize)
	previousMetadata := ""
	currentMetadata := ""
	var errs []error

	// Iterate over ResourceLogs
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)

		// iterate over InstrumentationLibraryLogs
		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)

			// iterate over Logs
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				currentMetadata = se.GetMetadata(log.Attributes())

				// If metadate differs from currently buffered, flush the buffer
				if currentMetadata != previousMetadata && previousMetadata != "" {
					se.sendAndPushErrors(&buffer, previousMetadata, &droppedTimeSeries, &errs)
				}

				// assign metadata
				previousMetadata = currentMetadata

				// add log to the buffer
				buffer = append(buffer, log)

				// Flush buffer to avoid overlow
				if len(buffer) == maxBufferSize {
					se.sendAndPushErrors(&buffer, previousMetadata, &droppedTimeSeries, &errs)
				}
			}
		}
	}

	// Flush pending logs
	se.sendAndPushErrors(&buffer, previousMetadata, &droppedTimeSeries, &errs)

	return droppedTimeSeries, componenterror.CombineErrors(errs)
}

func (se *sumologicexporter) sendLogs(buffer []pdata.LogRecord, fields string) error {
	body := strings.Builder{}

	if se.config.LogFormat == TextFormat {
		// Concatenate log lines using `\n`
		for j := 0; j < len(buffer); j++ {
			body.WriteString(buffer[j].Body().StringVal())
			if j == len(buffer)-1 {
				continue
			}
			body.WriteString("\n")
		}
	} else if se.config.LogFormat == JSONFormat {
		for j := 0; j < len(buffer); j++ {
			data := se.filterMetadata(buffer[j].Attributes(), true)
			data["log"] = buffer[j].Body().StringVal()
			jsonString, _ := json.Marshal(data)
			body.Write(jsonString)
			if j == len(buffer)-1 {
				continue
			}
			body.WriteString("\n")
		}
	} else {
		return errors.New("Unexpected log format")
	}

	return se.send(LogsPipeline, body.String(), fields)
}

// Send sends data to sumologic
func (se *sumologicexporter) send(pipeline string, body string, fields string) error {
	client := &http.Client{
		Timeout: se.config.TimeoutSettings.Timeout,
	}

	// Add headers
	req, _ := http.NewRequest("POST", se.config.URL, strings.NewReader(body))
	req.Header.Add("X-Sumo-Client", se.config.Client)

	if len(se.config.SourceHost) > 0 {
		req.Header.Add("X-Sumo-Host", se.config.SourceHost)
	}

	if len(se.config.SourceName) > 0 {
		req.Header.Add("X-Sumo-Name", se.config.SourceName)
	}

	if len(se.config.SourceCategory) > 0 {
		req.Header.Add("X-Sumo-Category", se.config.SourceCategory)
	}

	if pipeline == LogsPipeline {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Sumo-Fields", fields)
	} else if pipeline == MetricsPipeline {

	} else {
		return errors.New("Unexpected pipeline")
	}

	_, err := client.Do(req)

	// In case of error, push logs back to the channel
	if err != nil {
		fmt.Printf("Error during sending data to sumo: %q\n", err)
		return err
	}
	return nil
}
