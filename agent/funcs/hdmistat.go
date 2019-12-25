// Copyright 2017 Xiaomi, Inc.
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

package funcs

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"

	"github.com/open-falcon/falcon-plus/common/model"
)

func HdmiMetrics() []*model.MetricValue {
	state, _ := hdmiState()
	status, _ := hdmiStatus()
	if state && status {
		return []*model.MetricValue{GaugeValue("hdmi.connect", 1)}
	}

	return []*model.MetricValue{GaugeValue("hdmi.connect", 0)}
}

func hdmiStatus() (bool, error) {
	f := "/sys/devices/platform/display-subsystem/drm/card0/card0-HDMI-A-1/status"
	bs, err := ioutil.ReadFile(f)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))
	line, err := readLine(reader)
	if err == io.EOF {
		err = nil
		return false, err
	}
	if err != nil {
		return false, err
	}
	if "connected" == string(line) {
		return true, nil
	}

	return false, err
}

//bConnected bool, err error
func hdmiState() (bool, error) {
	f := "/sys/devices/virtual/switch/hdmi/state"
	bs, err := ioutil.ReadFile(f)
	if err != nil {
		return false, err
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))
	line, err := readLine(reader)
	if err == io.EOF {
		err = nil
		return false, err
	}
	if err != nil {
		return false, err
	}
	if "1" == string(line) {
		return true, nil
	}

	return false, err
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, isPrefix, err := r.ReadLine()
	for isPrefix && err == nil {
		var bs []byte
		bs, isPrefix, err = r.ReadLine()
		line = append(line, bs...)
	}

	return line, err
}
