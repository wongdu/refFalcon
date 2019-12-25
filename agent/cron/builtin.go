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

package cron

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/agent/funcs"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
)

var (
	bOfflineReboot       bool
	lastNotifyRebootTime string

	// 设置回调通知android端重启并修改次数
	cbFunc cbAddTimesFuncType

	// 当前是第几次断网重启
	currentRebootTimes int

	//当前启动是因为断网
	rebootCauseOffline bool
	// 开机时间
	bootUpTime string
)

func SetOfflineReboot(bReboot bool) {
	bOfflineReboot = bReboot
	log.Println("set offline reboot flag is: ", bReboot)
}

func SetLastNotifyRebootTime(paramRebootTime string) {
	lastNotifyRebootTime = paramRebootTime
}

type cbAddTimesFuncType func()

func SetCbFunc(cb cbAddTimesFuncType) {
	cbFunc = cb
}

func SetCurrentRebootTimes(currRebootTimes int) {
	if currRebootTimes > 0 {
		// 当前断网重启次数最小是1
		currentRebootTimes = currRebootTimes
	}
	log.Println("set current reboot times is: ", currRebootTimes)
}

func SetRebootCauseOffline(causeOffline bool) {
	rebootCauseOffline = causeOffline
	log.Println("set reboot cause offline flag is: ", causeOffline)
}

// 设置启动时间的时分秒即HH:mm:ss
func SetBootupTime(rbTime string) {
	bootUpTime = rbTime
	log.Println("set bootup time is: ", bootUpTime)
}

// 发送断网重启标记
func sendOfflineReboot(bFlag bool) {
	log.Println("send offline reboot bFlag is: ", bFlag)
	var mvs []*model.MetricValue
	if bFlag {
		tags := getRebootOnlineElapse(bootUpTime)
		log.Println("send offline reboot tags is: ", tags)
		mvs = []*model.MetricValue{funcs.GaugeValue("offline.reboot", 1, tags)}
		// mvs = []*model.MetricValue{funcs.GaugeValue("offline.reboot", 1, "a=1,b=2")}
	} else {
		mvs = []*model.MetricValue{funcs.GaugeValue("offline.reboot", 0)}
	}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)
}

func SyncBuiltinMetrics() {
	if g.Config().Heartbeat.Enabled && g.Config().Heartbeat.Addr != "" {
		go syncBuiltinMetrics()
	}
}

// 获取断网重启开机后到连上网络所经过的时间
func getRebootOnlineElapse(bootUpTime string) (strElapse string) {
	const TimeFormat = "2006-01-02 15:04:05"
	bootUpTime = time.Now().Format("2006-01-02") + " " + bootUpTime
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation(TimeFormat, bootUpTime, loc)
	TimeNow := time.Now()
	left := TimeNow.Sub(t)
	elapseSecond := int(left.Seconds())
	if elapseSecond <= 0 {
		return
	}

	return getTime(elapseSecond)
}

func getTime(second int) (strElapse string) {
	log.Println("get the elapse time is: ", second)
	var strTime string
	if second < 60 {
		strTime = fmt.Sprintf("%d秒", second)
	} else if second < 3600 {
		minute := second / 60
		second = second - minute*60
		if second == 0 {
			strTime = fmt.Sprintf("%d分", minute)
		} else {
			strTime = fmt.Sprintf("%d分%d秒", minute, second)
		}
	} else {
		hour := second / 3600
		minute := (second - hour*3600) / 60
		second = second - hour*3600 - minute*60
		strTime = fmt.Sprintf("%d小时%d分%d秒", hour, minute, second)
	}

	if currentRebootTimes > 0 {
		return fmt.Sprintf("offlineRebootInfo=第%d次断网重启，%s", currentRebootTimes, strTime)
	}
	return fmt.Sprintf("offlineRebootInfo=%s", strTime)
}

//判断当前次联网是否是因为刚刚断网重启
//也有可能就算断网重启成功了，依然是断网状态
// obsolete
func checkLastNotifyRebootTime(strRebootTime string) (bOfflineRebootNow bool) {
	//复用断网超时时间，当前时间跟底层上报断网重启时候的时间小于阈值就认为是断网重启
	if "" == strRebootTime {
		//如果上层传递的"上次断网重启时间"为空就直接返回
		return
	}

	const TimeFormat = "2006-01-02 15:04:05"
	offlineBaseTimeout := g.Config().OfflineBaseTimeout
	strRebootTime = time.Now().Format("2006-01-02") + " " + strRebootTime
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, _ := time.ParseInLocation(TimeFormat, strRebootTime, loc)
	elapse := time.Now().Unix() - t.Unix()
	if elapse > 0 && elapse < int64(offlineBaseTimeout) {
		return true
	}

	return
}

func syncBuiltinMetrics() {

	var timestamp int64 = -1
	var checksum string = "nil"

	duration := time.Duration(g.Config().Heartbeat.Interval) * time.Second

	offlineBaseTimeout := g.Config().OfflineBaseTimeout
	offlineRebootInterval := g.Config().OfflineRebootInterval
	var lastOnlineTime int64
	for {
		time.Sleep(duration)

		var ports = []int64{}
		var paths = []string{}
		var procs = make(map[string]map[int]string)
		var urls = make(map[string]string)

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		req := model.AgentHeartbeatRequest{
			Hostname: hostname,
			Checksum: checksum,
		}

		var resp model.BuiltinMetricResponse
		err = g.HbsClient.Call("Agent.BuiltinMetrics", req, &resp)
		if err != nil {
			log.Println("ERROR:", err)
			if bOfflineReboot {
				//可以断网重启的状态下，断网超时一端时间后发送通知给android层，然后等待重启
				// 当前断网重启次数-1之后是需要延时的重启间隔次数
				if time.Now().Unix()-lastOnlineTime > int64(offlineBaseTimeout+(currentRebootTimes-1)*offlineRebootInterval) {
					if cbFunc != nil {
						//已经断网了，发送不出去
						// sendOfflineReboot(true)
						cbFunc()
					}
					//通知一次到android层即可
					bOfflineReboot = false
				}
			}
			continue
		}
		lastOnlineTime = time.Now().Unix()
		//只有收到android端的上次重启时间才会进一步判断，防止浪费资源
		if bootUpTime != "" {
			//if checkLastNotifyRebootTime(lastNotifyRebootTime) {
			if rebootCauseOffline {
				// 如果是因为断网重启的，当连上网络后发送断网重启标记
				sendOfflineReboot(true)
			} else {
				//重启了，但不是断网重启，暂时没有用到
				sendOfflineReboot(false)
			}

			//只需要执行一次判断即可
			bootUpTime = ""
		}

		if resp.Timestamp <= timestamp {
			continue
		}

		if resp.Checksum == checksum {
			continue
		}

		timestamp = resp.Timestamp
		checksum = resp.Checksum

		for _, metric := range resp.Metrics {

			if metric.Metric == g.URL_CHECK_HEALTH {
				arr := strings.Split(metric.Tags, ",")
				if len(arr) != 2 {
					continue
				}
				url := strings.Split(arr[0], "=")
				if len(url) != 2 {
					continue
				}
				stime := strings.Split(arr[1], "=")
				if len(stime) != 2 {
					continue
				}
				if _, err := strconv.ParseInt(stime[1], 10, 64); err == nil {
					urls[url[1]] = stime[1]
				} else {
					log.Println("metric ParseInt timeout failed:", err)
				}
			}

			if metric.Metric == g.NET_PORT_LISTEN {
				arr := strings.Split(metric.Tags, "=")
				if len(arr) != 2 {
					continue
				}

				if port, err := strconv.ParseInt(arr[1], 10, 64); err == nil {
					ports = append(ports, port)
				} else {
					log.Println("metrics ParseInt failed:", err)
				}

				continue
			}

			if metric.Metric == g.DU_BS {
				arr := strings.Split(metric.Tags, "=")
				if len(arr) != 2 {
					continue
				}

				paths = append(paths, strings.TrimSpace(arr[1]))
				continue
			}

			if metric.Metric == g.PROC_NUM {
				arr := strings.Split(metric.Tags, ",")

				tmpMap := make(map[int]string)

				for i := 0; i < len(arr); i++ {
					if strings.HasPrefix(arr[i], "name=") {
						tmpMap[1] = strings.TrimSpace(arr[i][5:])
					} else if strings.HasPrefix(arr[i], "cmdline=") {
						tmpMap[2] = strings.TrimSpace(arr[i][8:])
					}
				}

				procs[metric.Tags] = tmpMap
			}
		}

		g.SetReportUrls(urls)
		g.SetReportPorts(ports)
		g.SetReportProcs(procs)
		g.SetDuPaths(paths)

	}
}
