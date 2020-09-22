// Copyright 2019 OpenTelemetry Authors
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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.uber.org/zap"
)

func TestExtractMetadata(t *testing.T) {
	cfg := &Config{}
	ttn := exportertest.NewNopLogsExporter()
	factory := NewFactory()
	processor, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, cfg, ttn)

	assert.NoError(t, err)

	attributes := pdata.NewAttributeMap()
	attributes.InsertString("fluent.tag", "var.log.containers.heapster-v1.5.2-58fdbb6f4d-vlc97_kube-system_heapster-b030cd9a8c10ea2dbd111db164118ec790df99857619e645d308c914bab7016c.log")
	processor.(*fluentbitk8sprocessor).ExtractMetadata(attributes)

	expected := make(map[string]string)
	expected["k8s.pod.name"] = "heapster-v1.5.2-58fdbb6f4d-vlc97"
	expected["pod_name"] = "heapster-v1.5.2-58fdbb6f4d-vlc97"
	expected["namespace"] = "kube-system"
	expected["container_name"] = "heapster"
	expected["docker_id"] = "b030cd9a8c10ea2dbd111db164118ec790df99857619e645d308c914bab7016c"

	for k, v := range expected {
		val, exists := attributes.Get(k)
		assert.True(t, exists)
		assert.Equal(t, pdata.AttributeValueSTRING, int(val.Type()))
		assert.Equal(t, v, val.StringVal())
	}
}
