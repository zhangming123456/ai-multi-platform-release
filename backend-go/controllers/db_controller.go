package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type DBController struct {
	BaseController
}

type sqlRequest struct {
	SQL      string `json:"sql"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

var selectRe = regexp.MustCompile(`(?i)^\s*(SELECT|PRAGMA|EXPLAIN)`)
var paginatedRe = regexp.MustCompile(`(?i)^\s*SELECT\b`)
var limitOffsetRe = regexp.MustCompile(`(?i)\s+LIMIT\s+(\d+)(\s+OFFSET\s+(\d+))?\s*;?\s*$`)

var timeColumnNames = map[string]bool{
	"created_at": true, "updated_at": true, "time": true, "date": true,
	"datetime": true, "timestamp": true, "create_time": true, "update_time": true,
	"created_time": true, "modified_at": true, "last_check_at": true,
	"token_expires_at": true, "publish_time": true, "generated_at": true,
}

func stripSQLLimitOffset(sql string) (string, bool, int, int) {
	m := limitOffsetRe.FindStringSubmatchIndex(sql)
	if m == nil {
		return sql, false, 0, 0
	}
	limit, _ := strconv.Atoi(sql[m[2]:m[3]])
	offset := 0
	if m[6] >= 0 {
		offset, _ = strconv.Atoi(sql[m[6]:m[7]])
	}
	stripped := strings.TrimSpace(sql[:m[0]] + sql[m[1]:])
	stripped = strings.TrimSuffix(stripped, ";")
	return strings.TrimSpace(stripped), true, limit, offset
}

// ListHistory GET /api/db/history
func (c *DBController) ListHistory() {
	if !c.CheckPermission("db_history:view:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	o := services.GetOrm()
	total, err := o.QueryTable(new(models.SqlHistory)).Count()
	if err != nil {
		total = 0
	}
	var rows []models.SqlHistory
	_, err = o.QueryTable(new(models.SqlHistory)).
		OrderBy("-created_at").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		All(&rows)
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询历史失败")
		return
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for i := range rows {
		items = append(items, map[string]interface{}{
			"id":         rows[i].ID,
			"username":   usernameByID(rows[i].UserID),
			"sql_text":   rows[i].SQLText,
			"is_success": rows[i].IsSuccess,
			"message":    rows[i].Message,
			"created_at": rows[i].CreatedAt.Format(isoFormat),
		})
	}
	c.OK(map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Execute POST /api/db/execute
func (c *DBController) Execute() {
	if !c.CheckPermission("db:execute:write") {
		return
	}
	user := c.CurrentUser()
	if user == nil {
		c.WriteError(http.StatusUnauthorized, "无法验证凭据")
		return
	}
	var req sqlRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	sql := strings.TrimSpace(req.SQL)
	if sql == "" {
		c.WriteError(http.StatusBadRequest, "SQL 命令不能为空")
		return
	}

	dangerousKeywords := []string{"DROP TABLE", "DROP DATABASE", "TRUNCATE", "ALTER TABLE", "ATTACH", "DETACH", "VACUUM", "REINDEX"}
	upperSQL := strings.ToUpper(sql)
	for _, keyword := range dangerousKeywords {
		if strings.Contains(upperSQL, keyword) {
			c.OK(map[string]interface{}{
				"success":  false,
				"message":  "禁止执行危险命令: " + keyword,
				"columns":  []string{},
				"rows":     [][]interface{}{},
				"is_query": false,
			})
			return
		}
	}
	if deleteRe.MatchString(sql) {
		c.OK(map[string]interface{}{
			"success":     false,
			"need_review": true,
			"change_type": "delete",
			"message":     "DELETE 删除操作需要提交审核，由至少 2 名审核员通过后自动执行",
		})
		return
	}
	if updateRe.MatchString(sql) {
		c.OK(map[string]interface{}{
			"success":     false,
			"need_review": true,
			"change_type": "update",
			"message":     "UPDATE 修改操作需要提交审核，由至少 2 名审核员通过后自动执行",
		})
		return
	}

	executedSQL := sql
	db, err := services.GetSQLDB()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "获取数据库连接失败")
		return
	}

	isQuery := selectRe.MatchString(sql)
	canPaginate := paginatedRe.MatchString(sql)

	var totalCount int64
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	message := ""
	columns := []string{}
	rowsData := [][]interface{}{}

	recordHistory := func(ok bool, msg string) {
		_, _ = services.GetOrm().Insert(&models.SqlHistory{
			ID:        newID(),
			UserID:    user.ID,
			SQLText:   executedSQL,
			IsSuccess: ok,
			Message:   msg,
		})
	}

	if canPaginate {
		noSemi := strings.TrimSuffix(strings.TrimSpace(sql), ";")
		base, _, _, _ := stripSQLLimitOffset(noSemi)
		if qerr := db.QueryRow("SELECT COUNT(*) FROM (" + base + ") AS _sub").Scan(&totalCount); qerr != nil {
			msg := "执行失败: " + qerr.Error()
			recordHistory(false, msg)
			c.OK(map[string]interface{}{"success": false, "message": msg, "columns": []string{}, "rows": [][]interface{}{}})
			return
		}
		page := req.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * pageSize
		executedSQL = fmt.Sprintf("SELECT * FROM (%s) AS _sub LIMIT %d OFFSET %d", base, pageSize, offset)
		columns, rowsData, err = queryRows(db, executedSQL)
		if err != nil {
			msg := "执行失败: " + err.Error()
			recordHistory(false, msg)
			c.OK(map[string]interface{}{"success": false, "message": msg, "columns": []string{}, "rows": [][]interface{}{}})
			return
		}
		curPage := 1
		if totalCount > 0 {
			curPage = page
		}
		message = fmt.Sprintf("查询成功，共 %d 行，当前第 %d 页", totalCount, curPage)
		sortRowsByTimeColumn(columns, rowsData)
		recordHistory(true, message)
		c.OK(map[string]interface{}{
			"success":     true,
			"message":     message,
			"columns":     columns,
			"rows":        rowsData,
			"row_count":   len(rowsData),
			"total_count": totalCount,
			"page_size":   pageSize,
			"is_query":    true,
		})
		return
	}

	if isQuery {
		columns, rowsData, err = queryRows(db, sql)
		if err != nil {
			msg := "执行失败: " + err.Error()
			recordHistory(false, msg)
			c.OK(map[string]interface{}{"success": false, "message": msg, "columns": []string{}, "rows": [][]interface{}{}})
			return
		}
		message = "查询成功"
		sortRowsByTimeColumn(columns, rowsData)
		recordHistory(true, message)
		c.OK(map[string]interface{}{
			"success":     true,
			"message":     message,
			"columns":     columns,
			"rows":        rowsData,
			"row_count":   len(rowsData),
			"total_count": 0,
			"page_size":   pageSize,
			"is_query":    true,
		})
		return
	}

	if _, err := db.Exec(sql); err != nil {
		msg := "执行失败: " + err.Error()
		recordHistory(false, msg)
		c.OK(map[string]interface{}{"success": false, "message": msg, "columns": []string{}, "rows": [][]interface{}{}})
		return
	}
	message = "命令执行成功"
	recordHistory(true, message)
	c.OK(map[string]interface{}{
		"success":  true,
		"message":  message,
		"columns":  []string{},
		"rows":     [][]interface{}{},
		"is_query": false,
	})
}

var deleteRe = regexp.MustCompile(`(?i)^\s*DELETE\b`)
var updateRe = regexp.MustCompile(`(?i)^\s*UPDATE\b`)

func queryRows(db *sql.DB, query string) ([]string, [][]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	cols := make([]string, len(columns))
	copy(cols, columns)

	data := make([][]interface{}, 0)
	for rows.Next() {
		vals := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		data = append(data, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return cols, data, nil
}

func sortRowsByTimeColumn(columns []string, rows [][]interface{}) {
	idx := -1
	for i, col := range columns {
		if timeColumnNames[strings.ToLower(col)] {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	sort.SliceStable(rows, func(a, b int) bool {
		return sortKey(rows[a][idx]) > sortKey(rows[b][idx])
	})
}

func sortKey(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
