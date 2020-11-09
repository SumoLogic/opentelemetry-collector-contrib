// Copyright 2020, OpenTelemetry Authors
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

package sumologicexporter

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressGzip(t *testing.T) {
	const message = "This is an example log"
	body := strings.NewReader(message)

	data, err := compressGZIP(body)
	require.NoError(t, err)

	assert.Equal(t, decodeGzip(t, data), message)
}

func decodeGzip(t *testing.T, data io.Reader) string {
	r, err := gzip.NewReader(data)
	require.NoError(t, err)

	buf := new(strings.Builder)
	_, err = io.Copy(buf, r)
	require.NoError(t, err)

	return buf.String()
}
func TestCompressDeflate(t *testing.T) {
	const message = "This is an example log"
	body := strings.NewReader(message)

	data, err := compressDeflate(body)
	require.NoError(t, err)

	assert.Equal(t, decodeDeflate(t, data), message)
}

func decodeDeflate(t *testing.T, data io.Reader) string {
	r := flate.NewReader(data)

	buf := new(strings.Builder)
	_, err := io.Copy(buf, r)
	require.NoError(t, err)

	return buf.String()
}
