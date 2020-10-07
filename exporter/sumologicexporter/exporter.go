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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type sumologicexporter struct {
	logger *zap.Logger
}

var _ (component.LogsExporter) = (*sumologicexporter)(nil)

func newLogsExporter(
	logger *zap.Logger,
) (component.LogsExporter, error) {
	se := &sumologicexporter{logger: logger}
	return se, nil
}

func newMetricsExporter(
	logger *zap.Logger,
) (component.MetricsExporter, error) {
	se := &sumologicexporter{logger: logger}
	return se, nil
}

func (se *sumologicexporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (se *sumologicexporter) Shutdown(context.Context) error {
	return nil
}

func (se *sumologicexporter) ConsumeLogs(_ context.Context, _ pdata.Logs) error {
	return nil
}

func (se *sumologicexporter) ConsumeMetrics(_ context.Context, _ pdata.Metrics) error {
	return nil
}
