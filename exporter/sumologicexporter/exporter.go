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

func (se *sumologicexporter) StartWorkers() {
	for i := 0; i < se.workers; i++ {
		go se.Sender()
	}
}

func (se *sumologicexporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (se *sumologicexporter) Shutdown(context.Context) error {
	close(se.ch)

	for {
		se.mux.Lock()
		if se.workers == 0 {
			se.mux.Unlock()
			return nil
		}
		se.mux.Unlock()
	}
}

func (se *sumologicexporter) Sender() {
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
		if len(buffer) == 100 {
			se.Send(buffer, previousMetadata)
			buffer = buffer[:0]
		}
	}
	se.Send(buffer, previousMetadata)

	se.mux.Lock()
	se.workers--
	se.mux.Unlock()
}

func (se *sumologicexporter) GetMetadata(attributes pdata.AttributeMap) string {
	buf := bytes.Buffer{}
	attributes.Sort().ForEach(func(k string, v pdata.AttributeValue) {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(v.StringVal())
		buf.WriteString(", ")
	})
	return buf.String()
}

func (se *sumologicexporter) Send(buffer []pdata.LogRecord, fields string) {
	client := &http.Client{}
	body := bytes.Buffer{}
	for j := 0; j < len(buffer); j++ {
		log := buffer[j]
		logBody := log.Body()
		body.WriteString(logBody.StringVal())
		body.WriteString("\n")
	}
	req, _ := http.NewRequest("POST", se.endpoint, bytes.NewBuffer(body.Bytes()))
	req.Header.Add("X-Sumo-Fields", fields)
	req.Header.Add("X-Sumo-Name", "otelcol")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	_, err := client.Do(req)

	if err != nil {
		fmt.Printf("Error during sending data to sumo: %q\n", err)
	}
}

// ConsumeLogs process logs and send them to the sumologic platform
func (se *sumologicexporter) ConsumeLogs(ctx context.Context, logs pdata.Logs) error {
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resource := logs.ResourceLogs().At(i)

		// ToDo: merge resources

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
