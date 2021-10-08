// Copyright The OpenTelemetry Authors
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

package filterprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/processor/filterlog"
)

type log struct {
	body               string
	severity           plog.SeverityNumber
	attributes         map[string]string
	resourceAttributes map[string]string
}

func logsToPdata(logs []log) plog.Logs {
	pl := plog.NewLogs()
	pl.ResourceLogs().EnsureCapacity(len(logs))

	for _, log := range logs {
		rl := pl.ResourceLogs().AppendEmpty()

		attrs := rl.Resource().Attributes()
		attrs.EnsureCapacity(len(log.resourceAttributes))
		for k, v := range log.resourceAttributes {
			attrs.InsertString(k, v)
		}
		// sort to ensure that assert.Equal works as expected
		attrs.Sort()

		rl.ScopeLogs().EnsureCapacity(1)
		l := rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		attrs = l.Attributes()
		for k, v := range log.attributes {
			attrs.InsertString(k, v)
		}
		// sort to ensure that assert.Equal works as expected
		attrs.Sort()

		l.SetSeverityNumber(log.severity)
		l.SetSeverityText(log.severity.String())
		l.Body().SetStringVal(log.body)
	}
	return pl
}

func TestLogExpr(t *testing.T) {

	type testCase struct {
		name        string
		expressions []string
		logs        []log
		expected    []log
	}

	allLogs := []log{
		// 0
		{
			body:     "Example log",
			severity: plog.SeverityNumberDEBUG,
			attributes: map[string]string{
				"foo":       "bar",
				"file.name": "first.log",
			},
			resourceAttributes: map[string]string{
				"app.name": "test",
				"instance": "pluto",
			},
		},
		// 1
		{
			body:     "[INFO] Another example log",
			severity: plog.SeverityNumberWARN,
			attributes: map[string]string{
				"foo":       "bar",
				"file.name": "second.log",
			},
			resourceAttributes: map[string]string{
				"app.name": "test",
				"instance": "mars",
			},
		},
		// 2
		{
			body:     "Info log",
			severity: plog.SeverityNumberINFO,
			attributes: map[string]string{
				"foo":       "bar",
				"file.name": "third.log",
			},
			resourceAttributes: map[string]string{
				"app.name": "test",
				"instance": "dione",
			},
		},
	}

	testCases := []testCase{
		{
			name: "match by body",
			expressions: []string{
				"Body matches 'Example'",
			},
			logs: allLogs,
			expected: []log{
				allLogs[0],
			},
		},
		{
			name: "match info logs",
			expressions: []string{
				"Body matches '[INFO]' || SeverityNumber == 9",
			},
			logs: allLogs,
			expected: []log{
				allLogs[1],
				allLogs[2],
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Logs: LogFilters{
					Include: &filterlog.LogMatchProperties{
						MatchType:   filterlog.Expr,
						Expressions: tc.expressions,
					},
				},
			}
			flp, err := newFilterLogsProcessor(&zap.Logger{}, cfg)

			expected := logsToPdata(tc.expected)
			logs := logsToPdata(tc.logs)

			require.NoError(t, err)
			result, err := flp.ProcessLogs(context.Background(), logs)

			require.NoError(t, err)
			assert.Equal(t, expected, result)
		})
	}
}
