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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type sumologicexporter struct {
	logger   *zap.Logger
	mux      sync.Mutex
	workers  int
	ch       chan pdata.LogRecord
	endpoint string
}

var _ (component.LogsExporter) = (*sumologicexporter)(nil)

// newLogsProcessor returns a component.LogsProcessor that extract k8s fields from the record tag
func newLogsExporter(
	logger *zap.Logger,
	endpoint string,
) (component.LogsExporter, error) {
	se := &sumologicexporter{logger: logger, workers: 5, ch: make(chan pdata.LogRecord), endpoint: endpoint}
	se.StartWorkers() // ToDo: move to Start (doesn't work for now)
	return se, nil
}

// StartWorkers create workers according to configuration
func (se *sumologicexporter) StartWorkers() {
	for i := 0; i < se.workers; i++ {
		go se.Sender()
	}
}

func (se *sumologicexporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (se *sumologicexporter) Shutdown(context.Context) error {
	// ToDo: Handle sending error while the channel is closed
	close(se.ch)

	// Wait for all workers to finish
	for {
		se.mux.Lock()
		if se.workers == 0 {
			se.mux.Unlock()
			return nil
		}
		se.mux.Unlock()
	}
}

// Sender creates a worker which is responsible for creating batches and sending data further
// LogRecords are grouped basing on the metadata set
func (se *sumologicexporter) Sender() {
	// Limit amount of logs in one batch to 100
	buffer := make([]pdata.LogRecord, 0, 100)
	previousMetadata := ""
	currentMetadata := ""

	for i := range se.ch {
		currentMetadata = se.GetMetadata(i.Attributes())
		// If the current and previous metadata differs, flush the buffer
		if currentMetadata != previousMetadata && previousMetadata != "" {
			se.Send(buffer, previousMetadata)
			buffer = buffer[:0]
		}
		previousMetadata = currentMetadata

		buffer = append(buffer, i)

		// Flush buffer, if it reaches the limit
		if len(buffer) == 100 {
			se.Send(buffer, previousMetadata)
			buffer = buffer[:0]
		}
	}

	// Send rest of data
	se.Send(buffer, previousMetadata)

	// Close the worker and decrement counter
	se.mux.Lock()
	se.workers--
	se.mux.Unlock()
}

func (se *sumologicexporter) GetMetadata(attributes pdata.AttributeMap) string {
	buf := strings.Builder{}
	i := 0
	attributes.Sort().ForEach(func(k string, v pdata.AttributeValue) {
		buf.WriteString(fmt.Sprintf("%s=%s", k, v.StringVal()))
		i++
		if i == attributes.Len() {
			return
		}
		buf.WriteString(", ")
	})
	return buf.String()
}

// Send sends data to sumologic
func (se *sumologicexporter) Send(buffer []pdata.LogRecord, fields string) {
	client := &http.Client{}
	body := bytes.Buffer{}

	// Concatenate log lines using `\n`
	for j := 0; j < len(buffer); j++ {
		log := buffer[j]
		logBody := log.Body()
		body.WriteString(logBody.StringVal())
		body.WriteString("\n")
	}

	// Add headers
	req, _ := http.NewRequest("POST", se.endpoint, bytes.NewBuffer(body.Bytes()))
	req.Header.Add("X-Sumo-Fields", fields)
	// ToDo: Make X-Sumo-Name configurable
	req.Header.Add("X-Sumo-Name", "otelcol")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err := client.Do(req)

	// In case of error, push logs back to the channel
	if err != nil {
		fmt.Printf("Error during sending data to sumo: %q\n", err)
		go se.PushBack(buffer)
	}
}

// PushBack pushes data back to the channel, so they can be processed again
func (se *sumologicexporter) PushBack(buffer []pdata.LogRecord) {
	for i := 0; i < len(buffer); i++ {
		// Put logs back to the channel
		se.ch <- buffer[i]
	}
}

// ConsumeLogs process logs and send them to the sumologic platform
func (se *sumologicexporter) ConsumeLogs(ctx context.Context, logs pdata.Logs) error {
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resource := logs.ResourceLogs().At(i)

		// ToDo: merge resources

		// Extract LogRecord and push it to the channel
		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				se.ch <- log
			}
		}
	}
	return nil
}
