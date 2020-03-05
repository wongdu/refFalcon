package resp_upper

import (
	"fmt"
	"time"

	"github.com/open-falcon/falcon-plus/common/model"
	"github.com/open-falcon/falcon-plus/modules/agent/funcs"
	"github.com/open-falcon/falcon-plus/modules/agent/g"
)

func UploadAlertCpuInfo(processName string, cpuValue float32) {
	var mvs []*model.MetricValue

	tags := fmt.Sprintf("packageName=%s", processName)
	mvs = []*model.MetricValue{funcs.GaugeValue("alert.cpu", cpuValue, tags)}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)
}

func UploadAlertMemoryInfo(processName string, memoryValue float32) {
	var mvs []*model.MetricValue

	tags := fmt.Sprintf("packageName=%s", processName)
	mvs = []*model.MetricValue{funcs.GaugeValue("alert.memory", memoryValue, tags)}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)

}

//tags可以包含包名，但是不能包含超时的时长，否则会一直告警，因为在组件里面judgeItemWithStrategy函数里，
//生成的model.Event的Id，既包含策略的id，也包含utils.PK(this.Endpoint, this.Metric, this.Tags),
//即tags不同认为是不用的event，会一直报警，超过管理平台配置的Max，即最大告警次数。
//20200304_10:33:29
//alertFlag:
//	1表示前台应用超时；
//	0表示无需告警，如果当前屏端有告警没有恢复则直接恢复；
//	-1表示没有正常获取到当前的前台应用，则只需告警一次；

func UploadForegroundAppTimeout(packageName string, alertFlag int) {
	var mvs []*model.MetricValue

	tags := fmt.Sprintf("packageName=%s", packageName)
	mvs = []*model.MetricValue{funcs.GaugeValue("foreground.app", alertFlag, tags)}

	hostname, _ := g.Hostname()
	now := time.Now().Unix()
	for j := 0; j < len(mvs); j++ {
		mvs[j].Step = int64(g.Config().Transfer.Interval)
		mvs[j].Endpoint = hostname
		mvs[j].Timestamp = now
	}

	g.SendToTransfer(mvs)
}
