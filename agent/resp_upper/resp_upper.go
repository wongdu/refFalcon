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
