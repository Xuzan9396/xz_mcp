package pgsql_db

import (
	"context"
	"testing"
	"time"
)

// 测试配置
var testConfig = PgConfig{
	Host:     "127.0.0.1",
	Port:     5432,
	User:     "xuzan",
	Password: "27252725",
	Database: "zero_web",
	SSLMode:  "disable",
}

func TestNewPgClient(t *testing.T) {
	client, err := NewPgClient(testConfig)
	if err != nil {
		t.Skipf("跳过测试 - 无法连接到PostgreSQL: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping失败: %v", err)
	}
}

func TestPgClient_GetInfo(t *testing.T) {
	client, err := NewPgClient(testConfig)
	if err != nil {
		t.Skipf("跳过测试 - 无法连接到PostgreSQL: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := client.GetInfo(ctx)
	if err != nil {
		t.Errorf("GetInfo失败: %v", err)
		return
	}

	if info["current_database"] == nil {
		t.Error("应该返回当前数据库信息")
	}

	if info["version"] == nil {
		t.Error("应该返回版本信息")
	}

	t.Logf("PostgreSQL信息: %+v", info)
}

func TestPgClient_ListSchemas(t *testing.T) {
	client, err := NewPgClient(testConfig)
	if err != nil {
		t.Skipf("跳过测试 - 无法连接到PostgreSQL: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.ListSchemas(ctx)
	if err != nil {
		t.Errorf("ListSchemas失败: %v", err)
		return
	}

	if result.Count == 0 {
		t.Error("应该至少有一个模式")
	}

	t.Logf("模式列表: %+v", result)
}

func TestPgClient_DDL_DML_Operations(t *testing.T) {
	client, err := NewPgClient(testConfig)
	if err != nil {
		t.Skipf("跳过测试 - 无法连接到PostgreSQL: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTableName := "test_mcp_table"

	// 1. 删除测试表（如果存在）
	dropSQL := "DROP TABLE IF EXISTS " + testTableName
	_, err = client.Exec(ctx, dropSQL)
	if err != nil {
		t.Errorf("删除测试表失败: %v", err)
		return
	}

	// 2. 创建测试表 (DDL)
	createSQL := `
		CREATE TABLE ` + testTableName + ` (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE,
			age INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	result, err := client.Exec(ctx, createSQL)
	if err != nil {
		t.Errorf("创建表失败: %v", err)
		return
	}
	t.Logf("创建表结果: %+v", result)

	// 3. 插入数据 (DML)
	insertSQL := "INSERT INTO " + testTableName + " (name, email, age) VALUES ($1, $2, $3) RETURNING id"
	insertResult, err := client.ExecWithLastInsertId(ctx, insertSQL, "张三", "zhangsan@example.com", 25)
	if err != nil {
		t.Errorf("插入数据失败: %v", err)
		return
	}
	t.Logf("插入数据结果: %+v", insertResult)

	// 4. 查询数据 (DML)
	selectSQL := "SELECT id, name, email, age FROM " + testTableName + " WHERE name = $1"
	queryResult, err := client.Query(ctx, selectSQL, "张三")
	if err != nil {
		t.Errorf("查询数据失败: %v", err)
		return
	}

	if queryResult.Count != 1 {
		t.Errorf("期望查询到1条记录，实际查询到%d条", queryResult.Count)
		return
	}
	t.Logf("查询数据结果: %+v", queryResult)

	// 5. 更新数据 (DML)
	updateSQL := "UPDATE " + testTableName + " SET age = $1 WHERE name = $2"
	updateResult, err := client.Exec(ctx, updateSQL, 26, "张三")
	if err != nil {
		t.Errorf("更新数据失败: %v", err)
		return
	}

	if updateResult.RowsAffected != 1 {
		t.Errorf("期望更新1行，实际更新%d行", updateResult.RowsAffected)
		return
	}
	t.Logf("更新数据结果: %+v", updateResult)

	// 6. 列出表信息
	tablesResult, err := client.ListTables(ctx, "public")
	if err != nil {
		t.Errorf("列出表失败: %v", err)
		return
	}
	t.Logf("表列表: %+v", tablesResult)

	// 7. 列出列信息
	columnsResult, err := client.ListColumns(ctx, testTableName, "public")
	if err != nil {
		t.Errorf("列出列失败: %v", err)
		return
	}

	if columnsResult.Count < 5 {
		t.Errorf("期望至少5个列，实际%d个", columnsResult.Count)
		return
	}
	t.Logf("列信息: %+v", columnsResult)

	// 8. 添加索引 (DDL)
	createIndexSQL := "CREATE INDEX idx_" + testTableName + "_email ON " + testTableName + " (email)"
	indexResult, err := client.Exec(ctx, createIndexSQL)
	if err != nil {
		t.Errorf("创建索引失败: %v", err)
		return
	}
	t.Logf("创建索引结果: %+v", indexResult)

	// 9. 列出索引信息
	indexesResult, err := client.ListIndexes(ctx, testTableName, "public")
	if err != nil {
		t.Errorf("列出索引失败: %v", err)
		return
	}

	if indexesResult.Count < 2 { // 至少有主键索引和我们创建的索引
		t.Errorf("期望至少2个索引，实际%d个", indexesResult.Count)
		return
	}
	t.Logf("索引信息: %+v", indexesResult)

	// 10. 删除数据 (DML)
	deleteSQL := "DELETE FROM " + testTableName + " WHERE name = $1"
	deleteResult, err := client.Exec(ctx, deleteSQL, "张三")
	if err != nil {
		t.Errorf("删除数据失败: %v", err)
		return
	}

	if deleteResult.RowsAffected != 1 {
		t.Errorf("期望删除1行，实际删除%d行", deleteResult.RowsAffected)
		return
	}
	t.Logf("删除数据结果: %+v", deleteResult)

	// 11. 验证数据已删除
	verifyResult, err := client.Query(ctx, "SELECT COUNT(*) as count FROM "+testTableName)
	if err != nil {
		t.Errorf("验证删除失败: %v", err)
		return
	}

	if len(verifyResult.Rows) > 0 {
		if count, ok := verifyResult.Rows[0]["count"].(int64); ok && count != 0 {
			t.Errorf("期望0条记录，实际%d条", count)
			return
		}
	}
	t.Logf("验证删除结果: %+v", verifyResult)

	// 12. 删除测试表 (DDL)
	_, err = client.Exec(ctx, "DROP TABLE "+testTableName)
	if err != nil {
		t.Errorf("删除测试表失败: %v", err)
		return
	}

	t.Log("PostgreSQL MCP测试完成！所有DDL和DML操作都成功执行")
}

func TestPgClient_TransactionSupport(t *testing.T) {
	client, err := NewPgClient(testConfig)
	if err != nil {
		t.Skipf("跳过测试 - 无法连接到PostgreSQL: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试事务支持
	tx, err := client.BeginTx(ctx)
	if err != nil {
		t.Errorf("开始事务失败: %v", err)
		return
	}

	// 在事务中创建临时表
	tempTableName := "temp_tx_test"
	createSQL := "CREATE TEMP TABLE " + tempTableName + " (id SERIAL PRIMARY KEY, value TEXT)"

	_, err = tx.ExecContext(ctx, createSQL)
	if err != nil {
		tx.Rollback()
		t.Errorf("在事务中创建表失败: %v", err)
		return
	}

	// 插入数据
	_, err = tx.ExecContext(ctx, "INSERT INTO "+tempTableName+" (value) VALUES ($1)", "test_value")
	if err != nil {
		tx.Rollback()
		t.Errorf("在事务中插入数据失败: %v", err)
		return
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		t.Errorf("提交事务失败: %v", err)
		return
	}

	t.Log("事务测试完成！")
}
