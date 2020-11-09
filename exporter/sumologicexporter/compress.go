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
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
)

func compressGZIP(data io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	var dataBytes bytes.Buffer

	dataBytes.ReadFrom(data)

	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(dataBytes.Bytes())
	if err != nil {
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func compressDeflate(data io.Reader) (io.Reader, error) {
	var buf bytes.Buffer
	var dataBytes bytes.Buffer

	dataBytes.ReadFrom(data)

	zw, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	_, err = zw.Write(dataBytes.Bytes())
	if err != nil {
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}
