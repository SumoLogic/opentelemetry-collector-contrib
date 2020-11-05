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
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type sumologicexporter struct {
	config *Config
	client *http.Client
	filter filter
}

func initExporter(cfg *Config) (*sumologicexporter, error) {
	switch cfg.LogFormat {
	case JSONFormat:
	case TextFormat:
	default:
		return nil, fmt.Errorf("unexpected log format: %s", cfg.LogFormat)
	}

	switch cfg.MetricFormat {
	case GraphiteFormat:
	case Carbon2Format:
	case PrometheusFormat:
	default:
		return nil, fmt.Errorf("unexpected metric format: %s", cfg.MetricFormat)
	}

	switch cfg.CompressEncoding {
	case GZIPCompression:
	case DeflateCompression:
	case NoCompression:
	default:
		return nil, fmt.Errorf("unexpected compression encoding: %s", cfg.CompressEncoding)
	}

	if len(cfg.HTTPClientSettings.Endpoint) == 0 {
		return nil, errors.New("endpoint is not set")
	}

	f, err := newFilter(cfg.MetadataAttributes)
	if err != nil {
		return nil, err
	}

	httpClient, err := cfg.HTTPClientSettings.ToClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP Client: %w", err)
	}

	se := &sumologicexporter{
		config: cfg,
		client: httpClient,
		filter: f,
	}

	return se, nil
}

func newLogsExporter(
	cfg *Config,
	params component.ExporterCreateParams,
) (component.LogsExporter, error) {
	se, err := initExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the logs exporter: %w", err)
	}

	return exporterhelper.NewLogsExporter(
		cfg,
		params.Logger,
		se.pushLogsData,
		// Disable exporterhelper Timeout, since we are using a custom mechanism
		// within exporter itself
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)
}

// pushLogsData groups data with common metadata and send them as separate batched requests
// it returns number of unsent logs and error which contains list of dropped records
// so they can be handle by the OTC retries mechanisms
func (se *sumologicexporter) pushLogsData(_ context.Context, ld pdata.Logs) (int, error) {
	var (
		currentMetadata  Fields
		previousMetadata Fields
		errors           []error
		sdr              = newSender(se.config, se.client, se.filter)
		droppedRecords   []pdata.LogRecord
	)

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
					dropped, err := sdr.sendLogs(previousMetadata)
					if err != nil {
						droppedRecords = append(droppedRecords, dropped...)
						errors = append(errors, err)
					}
					sdr.cleanBuffer()
				}

				// assign metadata
				previousMetadata = currentMetadata

				// add log to the buffer
				dropped, err := sdr.batch(log, previousMetadata)
				if err != nil {
					droppedRecords = append(droppedRecords, dropped...)
					errors = append(errors, err)
				}
			}
		}
	}

	// Flush pending logs
	dropped, err := sdr.sendLogs(previousMetadata)
	if err != nil {
		droppedRecords = append(droppedRecords, dropped...)
		errors = append(errors, err)
	}

	if len(droppedRecords) > 0 {
		// Move all dropped records to Logs
		droppedLogs := pdata.NewLogs()
		droppedLogs.ResourceLogs().Resize(1)
		droppedLogs.ResourceLogs().At(0).InstrumentationLibraryLogs().Resize(1)
		for _, record := range droppedRecords {
			droppedLogs.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().Append(record)
		}

		return len(droppedRecords), consumererror.PartialLogsError(componenterror.CombineErrors(errors), droppedLogs)
	}

	return 0, nil
}
