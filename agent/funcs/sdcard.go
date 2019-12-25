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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/toolkits_/file"
	"github.com/toolkits_/sys"
)

func SdcardMetrics() []*model.MetricValue {
	totalSpace, usedPercent, err := usedPercent()
	if err != nil {
		return []*model.MetricValue{GaugeValue("sdcard.warn", 1)}
	} else {
		return []*model.MetricValue{GaugeValue("sdcard.used", usedPercent, fmt.Sprintf("totalsize=%s", totalSpace))}
	}

}

func usedPercent() (string, float64, error) {
	var checkAbsPath string
	sdcardPath := "/sdcard"
	emulatedPath := "/storage/emulated/0"
	sdcardAbs, _ := filepath.Abs(sdcardPath)
	emulatedAbs, _ := filepath.Abs(emulatedPath)

	sdcardInfo, sdcardErr := os.Stat(sdcardAbs)
	emulatedInfo, emulatedErr := os.Stat(emulatedAbs)
	if sdcardErr != nil {
		return "", 0, sdcardErr
	}
	if emulatedErr != nil {
		return "", 0, emulatedErr
	}

	if false == sdcardInfo.IsDir() || false == emulatedInfo.IsDir() {
		return "", 0, errors.New("not directory")
	}

	if true == sdcardInfo.IsDir() {
		checkAbsPath = sdcardAbs
	} else if true == emulatedInfo.IsDir() {
		checkAbsPath = emulatedAbs
	} else {
		return "", 0, errors.New("not directory")
	}

	var bs []byte
	var cmdline string
	cmdline = fmt.Sprintf("df -h %s", checkAbsPath)
	bs, err := sys.CmdOutBytes("/system/bin/sh", "-c", cmdline)
	if err != nil {
		return "", 0, err
	}

	reader := bufio.NewReader(bytes.NewBuffer(bs))

	// ignore the first line
	line, e := file.ReadLine(reader)
	if e != nil {
		return "", 0, e
	}

	line, err = file.ReadLine(reader)
	if err != nil {
		return "", 0, err
	}
	fields := strings.Fields(string(line))
	percentUsed := fields[4]
	used, err := strconv.ParseFloat(percentUsed[:len(percentUsed)-1], 64)
	if err != nil {
		return "", 0, err
	}

	return fields[1], used, nil
}
