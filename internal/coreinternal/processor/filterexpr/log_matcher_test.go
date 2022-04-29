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

package filterexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestLogCompileExprError(t *testing.T) {
	_, err := NewMetricMatcher("")
	require.Error(t, err)
}

func TestLogRunExprError(t *testing.T) {
	matcher, err := NewLogMatcher("foo")
	require.NoError(t, err)
	matched, _ := matcher.match(logEnv{})
	require.False(t, matched)
}

func TestExpression(t *testing.T) {
	type testCase struct {
		name         string
		expression   string
		expected     bool
		body         pcommon.Value
		logName      string
		severity     int32
		severityText string
	}

	testCases := []testCase{
		{
			name:       "match body",
			expression: `Body matches 'my.log'`,
			expected:   true,
			body:       pcommon.NewValueString("my.log"),
		},
		{
			name:       "do not match body",
			expression: `Body matches 'my.log'`,
			expected:   false,
			body:       pcommon.NewValueString("mys.log"),
		},
		{
			name:       "match name",
			expression: `Name matches 'my l.g'`,
			expected:   true,
			body:       pcommon.NewValueEmpty(),
			logName:    "my log",
		},
		{
			name:       "do not match name",
			expression: `Name matches 'my l..g'`,
			expected:   false,
			body:       pcommon.NewValueEmpty(),
			logName:    "my log",
		},
		{
			name:       "match severity",
			expression: `SeverityNumber > 3`,
			expected:   true,
			body:       pcommon.NewValueEmpty(),
			severity:   5,
		},
		{
			name:       "do not match severity",
			expression: `SeverityNumber <= 3`,
			expected:   false,
			body:       pcommon.NewValueEmpty(),
			severity:   5,
		},
		{
			name:         "match severity name",
			expression:   `SeverityText matches 'foo'`,
			expected:     true,
			body:         pcommon.NewValueEmpty(),
			severityText: "foo bar",
		},
		{
			name:         "match severity name",
			expression:   `SeverityText matches 'foos'`,
			expected:     false,
			body:         pcommon.NewValueEmpty(),
			severityText: "foo bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewLogMatcher(tc.expression)
			require.NoError(t, err)
			l := plog.NewLogRecord()
			l.SetName(tc.logName)
			tc.body.CopyTo(l.Body())
			l.SetSeverityNumber(plog.SeverityNumber(tc.severity))
			l.SetSeverityText(tc.severityText)

			matched, err := matcher.MatchLog(l)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, matched)
		})
	}
}
