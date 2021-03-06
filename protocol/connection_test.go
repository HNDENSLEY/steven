// Copyright 2015 Matthew Collins
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

package protocol

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBasic(t *testing.T) {
	h := &Handshake{
		ProtocolVersion: 4,
		Host:            "",
		Port:            25565,
		Next:            1,
	}
	buf := &bytes.Buffer{}
	h.write(buf)

	h2 := &Handshake{}
	h2.read(bytes.NewReader(buf.Bytes()))

	if !reflect.DeepEqual(h, h2) {
		t.Fail()
	}
}
