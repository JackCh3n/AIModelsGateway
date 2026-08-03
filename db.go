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

	// 按日期
	rows, err = db.Query("SELECT date(timestamp),COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(total_tokens),0),COUNT(*) FROM usage_logs GROUP BY date(timestamp) ORDER BY date(timestamp) DESC")
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
		"totalInput":  totalInput,
		"totalOutput": totalOutput,
		"totalTokens": totalAll,
		"totalReqs":   totalReqs,
		"byProvider":  byProvider,
		"byModel":      byModel,
		"byDate":       byDate,
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
	}
}

func emptyStats() map[string]any {
	return map[string]any{
		"totalInput":  0,
		"totalOutput": 0,
		"totalTokens": 0,
		"totalReqs":   0,
		"byProvider":  map[string]map[string]int64{},
		"byModel":      map[string]map[string]int64{},
		"byDate":       map[string]map[string]int64{},
	}
}
