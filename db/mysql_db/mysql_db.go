package mysql_db

import (
	"fmt"
	"strings"
)

// QueryResult 查询结果
type QueryResult struct {
	Type       string                     `json:"type"`
	Data       []map[string]interface{}   `json:"data,omitempty"`
	ResultSets [][]map[string]interface{} `json:"result_sets,omitempty"` // 多结果集支持
	Count      int                        `json:"count,omitempty"`
	Success    bool                       `json:"success,omitempty"`
	Message    string                     `json:"message,omitempty"`
}

// ExecResult 执行结果
type ExecResult struct {
	Type         string `json:"type"`
	Success      bool   `json:"success"`
	RowsAffected int64  `json:"rows_affected,omitempty"`
	LastInsertID int64  `json:"last_insert_id,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Query 执行查询操作 (SELECT)
func Query(sql string, args ...interface{}) (*QueryResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	// 直接使用底层数据库连接进行查询，避免Base64编码问题
	rows, err := db.DB.Query(sql, args...)
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("failed to get columns: %v", err),
		}, nil
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &QueryResult{
				Type:    "error",
				Success: false,
				Message: fmt.Sprintf("failed to scan row: %v", err),
			}, nil
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("row iteration error: %v", err),
		}, nil
	}

	return &QueryResult{
		Type:    "select",
		Data:    results,
		Count:   len(results),
		Success: true,
	}, nil
}

// Exec 执行操作 (INSERT/UPDATE/DELETE)
func Exec(sql string, args ...interface{}) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	// 检查是否是修改操作
	sqlTrimmed := strings.TrimSpace(strings.ToUpper(sql))
	if strings.HasPrefix(sqlTrimmed, "INSERT") {
		// INSERT操作，获取插入ID
		lastID, err := db.ExecFindLastId(sql, args...)
		if err != nil {
			return &ExecResult{
				Type:    "error",
				Success: false,
				Message: err.Error(),
			}, nil
		}

		return &ExecResult{
			Type:         "insert",
			Success:      true,
			LastInsertID: lastID,
			RowsAffected: 1, // zmysql库的ExecFindLastId成功表示至少影响1行
		}, nil
	} else {
		// UPDATE/DELETE操作
		success, err := db.Exec(sql, args...)
		if err != nil {
			return &ExecResult{
				Type:    "error",
				Success: false,
				Message: err.Error(),
			}, nil
		}

		rowsAffected := int64(0)
		if success {
			rowsAffected = 1 // zmysql返回bool，true表示至少影响1行
		}

		return &ExecResult{
			Type:         "modification",
			Success:      success,
			RowsAffected: rowsAffected,
		}, nil
	}
}

// ExecWithLastID 执行INSERT操作并返回最后插入的ID
func ExecWithLastID(sql string, args ...interface{}) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	lastID, err := db.ExecFindLastId(sql, args...)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ExecResult{
		Type:         "insert",
		Success:      true,
		LastInsertID: lastID,
		RowsAffected: 1,
	}, nil
}

// CallProcedure 调用存储过程，支持动态数量的结果集
func CallProcedure(procName string, args ...interface{}) (*QueryResult, error) {
	if !IsConnected() || rawDB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// 使用通用的多结果集处理方式
	return callProcedureGeneric(procName, args...)
}

// callProcedureGeneric 通用的存储过程调用，支持任意数量的结果集
func callProcedureGeneric(procName string, args ...interface{}) (*QueryResult, error) {
	// 构建CALL语句
	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = "?"
	}
	sql := fmt.Sprintf("CALL %s(%s)", procName, strings.Join(placeholders, ","))

	// 使用原始的database/sql来处理多个结果集，绕过zmysql的限制
	rows, err := rawDB.Query(sql, args...)
	if err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer rows.Close()

	var allResultSets [][]map[string]interface{}

	// 处理多个结果集
	for {
		// 获取当前结果集的列信息
		columns, err := rows.Columns()
		if err != nil {
			// 如果无法获取列信息，可能是因为没有更多结果集
			if len(allResultSets) > 0 {
				break
			}
			return &QueryResult{
				Type:    "error",
				Success: false,
				Message: fmt.Sprintf("failed to get columns: %v", err),
			}, nil
		}

		// 读取当前结果集的所有行
		var currentResultSet []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return &QueryResult{
					Type:    "error",
					Success: false,
					Message: fmt.Sprintf("failed to scan row: %v", err),
				}, nil
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			currentResultSet = append(currentResultSet, row)
		}

		// 将当前结果集添加到所有结果集中
		allResultSets = append(allResultSets, currentResultSet)

		// 检查是否还有更多结果集
		if !rows.NextResultSet() {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return &QueryResult{
			Type:    "error",
			Success: false,
			Message: fmt.Sprintf("row iteration error: %v", err),
		}, nil
	}

	// 计算总记录数
	totalRecords := 0
	for _, resultSet := range allResultSets {
		totalRecords += len(resultSet)
	}

	// 根据结果集数量决定返回格式
	if len(allResultSets) == 1 {
		// 单结果集：使用data字段，确保ResultSets为nil
		return &QueryResult{
			Type:       "procedure",
			Data:       allResultSets[0],
			ResultSets: nil, // 显式设置为nil
			Count:      1,
			Success:    true,
			Message:    fmt.Sprintf("Successfully executed procedure with 1 result set. Total records: %d", totalRecords),
		}, nil
	} else {
		// 多结果集：使用result_sets字段，确保Data为nil
		return &QueryResult{
			Type:       "procedure",
			Data:       nil, // 显式设置为nil
			ResultSets: allResultSets,
			Count:      len(allResultSets),
			Success:    true,
			Message:    fmt.Sprintf("Successfully executed procedure with %d result sets. Total records: %d", len(allResultSets), totalRecords),
		}, nil
	}
}

// CreateProcedure 创建存储过程
func CreateProcedure(procedureSQL string) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	// 直接执行SQL，因为MySQL不支持在预处理语句中创建存储过程
	_, err := db.DB.Exec(procedureSQL)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ExecResult{
		Type:    "create_procedure",
		Success: true,
		Message: "Procedure created successfully",
	}, nil
}

// DropProcedure 删除存储过程
func DropProcedure(procName string) (*ExecResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	sql := fmt.Sprintf("DROP PROCEDURE IF EXISTS `%s`", procName)
	// 直接执行SQL，因为MySQL不支持在预处理语句中删除存储过程
	_, err := db.DB.Exec(sql)
	if err != nil {
		return &ExecResult{
			Type:    "error",
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ExecResult{
		Type:    "drop_procedure",
		Success: true,
		Message: fmt.Sprintf("Procedure %s dropped successfully", procName),
	}, nil
}

// ShowProcedures 显示存储过程列表
func ShowProcedures(databaseName string) (*QueryResult, error) {
	if !IsConnected() {
		return nil, fmt.Errorf("database not connected")
	}

	sql := "SELECT ROUTINE_NAME as name, ROUTINE_TYPE as type, CREATED as created, LAST_ALTERED as last_altered FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_SCHEMA = ?"
	return Query(sql, databaseName)
}
