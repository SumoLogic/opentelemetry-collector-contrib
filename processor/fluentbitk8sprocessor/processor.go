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

package fluentbitk8sprocessor

import (
	"context"
	"regexp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type fluentbitk8sprocessor struct {
	logger           *zap.Logger
	nextLogsConsumer consumer.LogsConsumer
}

var _ (component.LogsProcessor) = (*fluentbitk8sprocessor)(nil)

// newLogsProcessor returns a component.LogsProcessor that extract k8s fields from the record tag
func newLogsProcessor(
	logger *zap.Logger,
	nextLogsConsumer consumer.LogsConsumer,
) (component.LogsProcessor, error) {
	fbkp := &fluentbitk8sprocessor{logger: logger, nextLogsConsumer: nextLogsConsumer}
	return fbkp, nil
}

func (fbkp *fluentbitk8sprocessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (fbkp *fluentbitk8sprocessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (fbkp *fluentbitk8sprocessor) Shutdown(context.Context) error {
	return nil
}

func (fbkp *fluentbitk8sprocessor) ExtractMetadata(attributes pdata.AttributeMap) {
	attr, exists := attributes.Get("fluent.tag")

	if exists && attr.Type() == pdata.AttributeValueSTRING {
		tag := attr.StringVal()
		regex := regexp.MustCompile(`\.containers\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\.log$`)
		details := regex.FindAllStringSubmatch(tag, -1)

		if len(details) == 1 && len(details[0]) == 5 {
			attributes.InsertString("k8s.pod.name", details[0][1])
			attributes.InsertString("pod_name", details[0][1])
			attributes.InsertString("namespace", details[0][2])
			attributes.InsertString("container_name", details[0][3])
			attributes.InsertString("docker_id", details[0][4])
		}
	}
}

// ConsumeLogs process logs and convberts fluentbit k8s tag to metadata
func (fbkp *fluentbitk8sprocessor) ConsumeLogs(ctx context.Context, logs pdata.Logs) error {
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		resource := logs.ResourceLogs().At(i)

		// ToDo: fluent bit tag can potantially be in resource attributes

		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				fbkp.ExtractMetadata(log.Attributes())
			}
		}
	}
	return fbkp.nextLogsConsumer.ConsumeLogs(ctx, logs)
}
