package sb_cron

import (
	"github.com/robfig/cron"
	"gitlab.com/stihi/stihi-backend/cache_level2"
)

var currentCron *cron.Cron

func Init(dbConn *cache_level2.CacheLevel2) {
	currentCron = cron.New()

	_ = currentCron.AddFunc("0 * * * *", dbConn.SyncArticles)

	currentCron.Start()
}

func Stop() {
	currentCron.Stop()
}
