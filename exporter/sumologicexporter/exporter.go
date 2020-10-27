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
	"net/http"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	logKey        string = "log"
	maxBufferSize int    = 100
)

type sumologicexporter struct {
	config *Config
	client *http.Client
}

func newLogsExporter(
	cfg *Config,
) (component.LogsExporter, error) {
	se, err := initExporter(cfg)
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

func initExporter(cfg *Config) (*sumologicexporter, error) {
	se := &sumologicexporter{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.TimeoutSettings.Timeout,
		},
	}

	return se, nil
}

// pushLogsData groups data with common metadata uses sendAndPushErrors to send data to sumologic
func (se *sumologicexporter) pushLogsData(ctx context.Context, ld pdata.Logs) (droppedTimeSeries int, err error) {
	var (
		currentMetadata  FieldsType
		previousMetadata FieldsType
		errors           []error
	)

	filter, err := newFiltering(se.config.MetadataFields)
	if err != nil {
		return 0, err
	}

	sdr := newSender(se.config, se.client, filter)

	// Iterate over ResourceLogs
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)

		// iterate over InstrumentationLibraryLogs
		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)

			// iterate over Logs
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				currentMetadata = sdr.filter.GetMetadata(log.Attributes())

				// If metadata differs from currently buffered, flush the buffer
				if currentMetadata != previousMetadata && previousMetadata != "" {
					sdr.sendAndPushErrors(previousMetadata, &droppedTimeSeries, &errors)
				}

				// assign metadata
				previousMetadata = currentMetadata

				// add log to the buffer
				sdr.buffer = append(sdr.buffer, log)

				// Flush buffer to avoid overlow
				if len(sdr.buffer) == maxBufferSize {
					sdr.sendAndPushErrors(previousMetadata, &droppedTimeSeries, &errors)
				}
			}
		}
	}

	// Flush pending logs
	sdr.sendAndPushErrors(previousMetadata, &droppedTimeSeries, &errors)

	return droppedTimeSeries, componenterror.CombineErrors(errors)
}
