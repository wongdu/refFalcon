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

package agent

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-falcon/falcon-plus/modules/agent/cron"
	"github.com/open-falcon/falcon-plus/modules/agent/funcs"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
	"github.com/open-falcon/falcon-plus/modules/agent/http"
	respUpper "github.com/open-falcon/falcon-plus/modules/agent/resp_upper"
	"github.com/open-falcon/falcon-plus/modules/agent/watch"
)

const (
	Watch_Dir = "/data/system/dropbox/"
)

func agentMain(cfgPath string) {
	// cfg := flag.String("c", "/sdcard/imprexion/cfg.json", "configuration file")
	// cfg := flag.String("c", "/sdcard/imprexion/config/com.imprexion.service.falcon.agent/cfg.json", "configuration file")
	cfg := flag.String("c", cfgPath, "configuration file")
	version := flag.Bool("v", false, "show version")
	check := flag.Bool("check", false, "check collector")

	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	if *check {
		funcs.CheckCollector()
		os.Exit(0)
	}

	g.ParseConfig(*cfg)

	if g.Config().Debug {
		g.InitLog("debug")
	} else {
		g.InitLog("info")
	}

	g.InitRootDir()
	g.InitLocalIp()
	g.InitRpcClients()

	funcs.BuildMappers()

	cron.SetCbFunc(cbAddOfflineRebootTimes)
	go cron.InitDataHistory()

	cron.ReportAgentStatus()
	cron.SyncMinePlugins()
	cron.SyncBuiltinMetrics()
	cron.SyncTrustableIps()
	cron.Collect()
	// go watch.WatchDir(Watch_Dir)
	watch.SetCbFunc(cbChangeModeBits)
	go watch.ProWatchDir()

	go http.Start()

	select {}
}

// 把配置文件的路径传到底层
func AgentEnter(cfgPath string) {
	go agentMain(cfgPath)
}

// obsolete
func TransferName(fileName string) {
	// watch.TransferName(fileName)
}

func SetOfflineReboot(bReboot bool) {
	cron.SetOfflineReboot(bReboot)
}

func SetLastNotifyRebootTime(rebootTime string) {
	cron.SetLastNotifyRebootTime(rebootTime)
}

//将新的增量文件上传到阿里云并且发送metric
func NotifyIncrementFile(fileName string) {
	watch.NotifyIncrementFile(fileName)
}

//设置当前是一天中第几次断网重启
func SetCurrentRebootTimes(currRebootTimes int) {
	cron.SetCurrentRebootTimes(currRebootTimes)
}

//设置是否是断网重启
func SetRebootCauseOffline(rebootCauseOffline bool) {
	cron.SetRebootCauseOffline(rebootCauseOffline)
}

// 设置开机时间
func SetBootupTime(rebootTime string) {
	cron.SetBootupTime(rebootTime)
}

// 上报cpu使用率
func UploadAlertCpuInfo(processName string, cpuValue float32) {
	respUpper.UploadAlertCpuInfo(processName, cpuValue)
}

// 上报内存使用率
func UploadAlertMemoryInfo(processName string, memoryValue float32) {
	respUpper.UploadAlertMemoryInfo(processName, memoryValue)
}

// 上报前台应用超时
func UploadForegroundAppTimeout(packageName string, alertFlag int) {
	respUpper.UploadForegroundAppTimeout(packageName, alertFlag)
}
