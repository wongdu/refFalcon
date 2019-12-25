package event

import (
	"database/sql"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/astaxie/beego/orm"

	coommonModel "github.com/open-falcon/falcon-plus/common/model"
)

// const timeLayout = "2006-01-02 15:04:05"
// 该常量在当前包的event_operation.go中已经声明了，
// 所以当前文件直接使用无需重复声明，否则会报错"timeLayout redeclared in this block"

func InsertAlert(eve *coommonModel.Event) (res sql.Result, err error) {
	q := orm.NewOrm()

	sqltemplete := `INSERT INTO alerts (
        event_caseId,
        endpoint,
        note,
        tpl_creator,
        timestamp
    ) VALUES(?,?,?,?,?)`

	tpl_creator := ""
	if eve.Tpl() != nil {
		tpl_creator = eve.Tpl().Creator
	}

	res, err = q.Raw(
		sqltemplete,
		eve.Id,
		eve.Endpoint,
		eve.Note(),
		tpl_creator,
		time.Unix(eve.EventTime, 0).Format(timeLayout),
	).Exec()

	if err != nil {
		log.Errorf("insert event to db fail, error:%v", err)
	} else {
		lastid, _ := res.LastInsertId()
		log.Debug("insert event to db succ, last_insert_id:", lastid)
	}
	return
}

func DeleteExpiredAlert(before time.Time, limit int) {
	t := before.Format(timeLayout)
	sqlTpl := `delete from alerts where timestamp<? limit ?`
	q := orm.NewOrm()
	resp, err := q.Raw(sqlTpl, t, limit).Exec()
	if err != nil {
		log.Errorf("delete alert older than %v fail, error:%v", t, err)
	} else {
		affected, _ := resp.RowsAffected()
		log.Debugf("delete alert older than %v, rows affected:%v", t, affected)
	}
}
