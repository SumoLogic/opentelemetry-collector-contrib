// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cascadingfilterprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/cascadingfilterprocessor/config"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig().(*config.Config)
	// Manually set required fields
	cfg.ExpectedNewTracesPerSec = 64
	cfg.PolicyCfgs = []config.PolicyCfg{
		{
			Name: "test-policy",
		},
	}

	params := component.ProcessorCreateParams{Logger: zap.NewNop()}
	tp, err := factory.CreateTracesProcessor(context.Background(), params, cfg, consumertest.NewNop())
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create trace processor")
}
