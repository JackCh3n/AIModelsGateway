package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	db     *sql.DB
	dbOnce sync.Once
	dbPath string
)

func initDB() {
	dbOnce.Do(func() {
		exe, _ := os.Executable()
		dataDir := filepath.Join(filepath.Dir(exe), "data")
		os.MkdirAll(dataDir, 0755)
		dbPath = filepath.Join(dataDir, "usage.db")
		d, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
		if err != nil {
			log.Printf("sqlite open error: %v", err)
			return
		}
		// 连接池配置：WAL 模式下读写不互斥，可支持较高并发
		d.SetMaxOpenConns(50)
		d.SetMaxIdleConns(10)
		d.SetConnMaxLifetime(30 * time.Minute)
		db = d
		// 创建表
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS usage_logs (
			id TEXT PRIMARY KEY,
			provider_id TEXT,
			provider_name TEXT,
			model TEXT,
			input_tokens INTEGER,
			output_tokens INTEGER,
			total_tokens INTEGER,
			client_format TEXT,
			timestamp DATETIME
		)`)
		if err != nil {
			log.Printf("sqlite create table error: %v", err)
			return
		}
		// 创建索引加速查询
		_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_ts ON usage_logs(timestamp DESC)")
		_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_provider ON usage_logs(provider_name)")
		_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_model ON usage_logs(model)")
		// 错误日志表
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS error_logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME,
			status_code INTEGER,
			provider_id TEXT,
			provider_name TEXT,
			model TEXT,
			route TEXT,
			message TEXT
		)`)
		if err != nil {
			log.Printf("sqlite create error_logs table error: %v", err)
		}
		_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_errlogs_ts ON error_logs(timestamp DESC)")
		// 启动定时清理（保留30天）
		go cleanupOldLogs()
	})
}

// dbAddUsageLog 写入一条用量日志
func dbAddUsageLog(entry UsageLog) {
	initDB()
	if db == nil {
		return
	}
	_, err := db.Exec("INSERT OR IGNORE INTO usage_logs(id,provider_id,provider_name,model,input_tokens,output_tokens,total_tokens,client_format,timestamp) VALUES(?,?,?,?,?,?,?,?,?)",
		entry.ID, entry.ProviderID, entry.ProviderName, entry.Model,
		entry.InputTokens, entry.OutputTokens, entry.TotalTokens, entry.ClientFormat, entry.Timestamp,
	)
	if err != nil {
		log.Printf("sqlite insert error: %v", err)
	}
}

// dbAddErrorLog 写入一条错误日志
func dbAddErrorLog(entry ErrorLog) {
	initDB()
	if db == nil {
		return
	}
	_, err := db.Exec("INSERT OR IGNORE INTO error_logs(id,timestamp,status_code,provider_id,provider_name,model,route,message) VALUES(?,?,?,?,?,?,?,?)",
		entry.ID, entry.Timestamp, entry.StatusCode, entry.ProviderID, entry.ProviderName, entry.Model, entry.Route, entry.Message,
	)
	if err != nil {
		log.Printf("sqlite insert error_log error: %v", err)
	}
}

// dbGetErrorLogs 分页查询错误日志（最新在前）
func dbGetErrorLogs(page, pageSize int) ([]ErrorLog, int) {
	initDB()
	if db == nil {
		return []ErrorLog{}, 0
	}
	var total int
	_ = db.QueryRow("SELECT COUNT(*) FROM error_logs").Scan(&total)
	off := (page - 1) * pageSize
	rows, err := db.Query("SELECT id,timestamp,status_code,provider_id,provider_name,model,route,message FROM error_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?", pageSize, off)
	if err != nil {
		return []ErrorLog{}, total
	}
	defer rows.Close()
	var list []ErrorLog
	for rows.Next() {
		var e ErrorLog
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.StatusCode, &e.ProviderID, &e.ProviderName, &e.Model, &e.Route, &e.Message); err == nil {
			list = append(list, e)
		}
	}
	if list == nil {
		list = []ErrorLog{}
	}
	return list, total
}

// dbBatchInsertUsageLogs 批量写入用量日志，减少事务开销提升吞吐
func dbBatchInsertUsageLogs(entries []UsageLog) {
	initDB()
	if db == nil || len(entries) == 0 {
		return
	}
	tx, err := db.Begin()
	if err != nil {
		log.Printf("sqlite batch begin tx error: %v", err)
		// 降级为逐条写入
		for _, e := range entries {
			dbAddUsageLog(e)
		}
		return
	}
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO usage_logs(id,provider_id,provider_name,model,input_tokens,output_tokens,total_tokens,client_format,timestamp) VALUES(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		tx.Rollback()
		log.Printf("sqlite batch prepare error: %v", err)
		return
	}
	defer stmt.Close()
	for _, e := range entries {
		_, _ = stmt.Exec(e.ID, e.ProviderID, e.ProviderName, e.Model,
			e.InputTokens, e.OutputTokens, e.TotalTokens, e.ClientFormat, e.Timestamp)
	}
	if err := tx.Commit(); err != nil {
		log.Printf("sqlite batch commit error: %v", err)
	}
}

// dbGetUsageStats 统计用量
func dbGetUsageStats() map[string]any {
	initDB()
	if db == nil {
		return emptyStats()
	}
	// 总计
	var totalInput, totalOutput, totalAll, totalReqs int64
	row := db.QueryRow("SELECT COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(total_tokens),0),COUNT(*) FROM usage_logs")
	row.Scan(&totalInput, &totalOutput, &totalAll, &totalReqs)

	byProvider := map[string]map[string]int64{}
	byModel := map[string]map[string]int64{}
	byDate := map[string]map[string]int64{}
	today := time.Now().Format("2006-01-02")
	byModelToday := map[string]int64{}

	// 按站点
	rows, err := db.Query("SELECT provider_name,COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(total_tokens),0),COUNT(*) FROM usage_logs GROUP BY provider_name")
	if err == nil {
		for rows.Next() {
			var name string
			var input, output, total, count int64
			rows.Scan(&name, &input, &output, &total, &count)
			byProvider[name] = map[string]int64{"input": input, "output": output, "total": total, "count": count}
		}
		rows.Close()
	}

	// 按模型
	rows, err = db.Query("SELECT model,COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(total_tokens),0),COUNT(*) FROM usage_logs GROUP BY model")
	if err == nil {
		for rows.Next() {
			var model string
			var input, output, total, count int64
			rows.Scan(&model, &input, &output, &total, &count)
			byModel[model] = map[string]int64{"input": input, "output": output, "total": total, "count": count}
		}
		rows.Close()
	}

	// 当日各模型请求次数
	rows, err = db.Query("SELECT model,COUNT(*) FROM usage_logs WHERE substr(timestamp,1,10)=? GROUP BY model ORDER BY COUNT(*) DESC", today)
	if err == nil {
		for rows.Next() {
			var model string
			var count int64
			rows.Scan(&model, &count)
			byModelToday[model] = count
		}
		rows.Close()
	}

	// 按日期
	// 注意：SQLite 的 date() 函数无法解析带时区偏移和纳秒的时间戳格式
	// (如 2026-08-05T01:10:41.8356031+08:00)，会返回 NULL。
	// 因此改用 substr 提取前 10 位日期 (YYYY-MM-DD)。
	rows, err = db.Query("SELECT substr(timestamp,1,10),COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(total_tokens),0),COUNT(*) FROM usage_logs GROUP BY substr(timestamp,1,10) ORDER BY substr(timestamp,1,10) DESC")
	if err == nil {
		for rows.Next() {
			var date string
			var input, output, total, count int64
			rows.Scan(&date, &input, &output, &total, &count)
			byDate[date] = map[string]int64{"input": input, "output": output, "total": total, "count": count}
		}
		rows.Close()
	}

	return map[string]any{
		"totalInput":   totalInput,
		"totalOutput":  totalOutput,
		"totalTokens":  totalAll,
		"totalReqs":    totalReqs,
		"byProvider":   byProvider,
		"byModel":      byModel,
		"byDate":       byDate,
		"byModelToday": byModelToday,
	}
}

// dbGetRecentLogs 分页查询日志（最新在前）
func dbGetRecentLogs(page, pageSize int) ([]UsageLog, int) {
	initDB()
	if db == nil {
		return []UsageLog{}, 0
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 50
	}
	var total int
	db.QueryRow("SELECT COUNT(*) FROM usage_logs").Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := db.Query("SELECT id,provider_id,provider_name,model,input_tokens,output_tokens,total_tokens,client_format,timestamp FROM usage_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?", pageSize, offset)
	if err != nil {
		log.Printf("sqlite query logs error: %v", err)
		return []UsageLog{}, total
	}
	defer rows.Close()

	result := []UsageLog{}
	for rows.Next() {
		var l UsageLog
		var ts time.Time
		rows.Scan(&l.ID, &l.ProviderID, &l.ProviderName, &l.Model, &l.InputTokens, &l.OutputTokens, &l.TotalTokens, &l.ClientFormat, &ts)
		l.Timestamp = ts
		result = append(result, l)
	}
	return result, total
}

// dbClearLogs 清空所有日志
func dbClearLogs() {
	initDB()
	if db == nil {
		return
	}
	_, err := db.Exec("DELETE FROM usage_logs")
	if err != nil {
		log.Printf("sqlite clear logs error: %v", err)
	}
	// 重置自增
	_, _ = db.Exec("VACUUM")
}

// cleanupOldLogs 定时清理30天前的日志
func cleanupOldLogs() {
	for {
		time.Sleep(1 * time.Hour)
		if db == nil {
			continue
		}
		cutoff := time.Now().AddDate(0, 0, -30)
		_, err := db.Exec("DELETE FROM usage_logs WHERE timestamp < ?", cutoff)
		if err != nil {
			log.Printf("sqlite cleanup error: %v", err)
		}
		_, err = db.Exec("DELETE FROM error_logs WHERE timestamp < ?", cutoff)
		if err != nil {
			log.Printf("sqlite cleanup error_logs error: %v", err)
		}
	}
}

func emptyStats() map[string]any {
	return map[string]any{
		"totalInput":  0,
		"totalOutput": 0,
		"totalTokens": 0,
		"totalReqs":   0,
		"byProvider":  map[string]map[string]int64{},
		"byModel":     map[string]map[string]int64{},
		"byDate":      map[string]map[string]int64{},
	}
}
