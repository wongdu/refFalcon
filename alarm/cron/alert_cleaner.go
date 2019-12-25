package cron

import (
	"time"

	"github.com/open-falcon/falcon-plus/modules/alarm/g"
	eventmodel "github.com/open-falcon/falcon-plus/modules/alarm/model/event"
)

func CleanExpiredAlert() {
	for {

		retention_days := g.Config().Housekeeper.EventRetentionDays
		delete_batch := g.Config().Housekeeper.EventDeleteBatch

		now := time.Now()
		before := now.Add(time.Duration(-retention_days*24) * time.Hour)
		eventmodel.DeleteExpiredAlert(before, delete_batch)

		time.Sleep(time.Second * 60)
	}
}
