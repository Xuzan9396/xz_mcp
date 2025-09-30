package mysql_db

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// 测试配置
var testConfig = ConnectionConfig{
	Username:        "root",
	Password:        "123456",
	Addr:            "127.0.0.1:3326",
	DatabaseName:    "test_db",
	Debug:           true,
	MaxOpenConns:    10,
	MaxIdleConns:    5,
	ConnMaxLifetime: 1 * time.Hour,
}

func TestInitDB(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	if !IsConnected() {
		t.Fatal("数据库应该已连接")
	}
}

func TestCreateTable(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 删除测试表（如果存在）
	DropTable("test_users")

	// 创建测试表
	createSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	result, err := CreateTable(createSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	if !result.Success {
		t.Fatalf("创建表应该成功，得到结果: %+v", result)
	}

	t.Logf("创建表成功: %+v", result)
}

func TestInsertAndQuery(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 删除测试表（如果存在）
	DropTable("test_users")

	// 创建测试表
	createSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err = CreateTable(createSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	// 插入测试数据
	insertSQL := "INSERT INTO test_users (name, email, age) VALUES (?, ?, ?)"
	execResult, err := ExecWithLastID(insertSQL, "张三", "zhangsan@test.com", 25)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	if !execResult.Success {
		t.Fatalf("插入数据应该成功，得到结果: %+v", execResult)
	}

	if execResult.LastInsertID <= 0 {
		t.Fatalf("应该返回有效的插入ID，得到: %d", execResult.LastInsertID)
	}

	t.Logf("插入数据成功: %+v", execResult)

	// 查询数据
	querySQL := "SELECT * FROM test_users WHERE id = ?"
	queryResult, err := Query(querySQL, execResult.LastInsertID)
	if err != nil {
		t.Fatalf("查询数据失败: %v", err)
	}

	if !queryResult.Success {
		t.Fatalf("查询数据应该成功，得到结果: %+v", queryResult)
	}

	if queryResult.Count != 1 {
		t.Fatalf("应该查询到1条记录，得到: %d", queryResult.Count)
	}

	t.Logf("查询数据成功: %+v", queryResult)

	// 验证数据内容
	user := queryResult.Data[0]
	if user["name"] != "张三" {
		t.Fatalf("用户名应该是'张三'，得到: %v", user["name"])
	}
	if user["email"] != "zhangsan@test.com" {
		t.Fatalf("邮箱应该是'zhangsan@test.com'，得到: %v", user["email"])
	}
}

func TestUpdateAndDelete(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 删除测试表（如果存在）
	DropTable("test_users")

	// 创建测试表
	createSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err = CreateTable(createSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	// 插入测试数据
	insertSQL := "INSERT INTO test_users (name, email, age) VALUES (?, ?, ?)"
	_, err = ExecWithLastID(insertSQL, "张三", "zhangsan@test.com", 25)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 更新数据
	updateSQL := "UPDATE test_users SET age = ? WHERE name = ?"
	updateResult, err := Exec(updateSQL, 30, "张三")
	if err != nil {
		t.Fatalf("更新数据失败: %v", err)
	}

	if !updateResult.Success {
		t.Fatalf("更新数据应该成功，得到结果: %+v", updateResult)
	}

	t.Logf("更新数据成功: %+v", updateResult)

	// 验证更新结果
	queryResult, err := Query("SELECT age FROM test_users WHERE name = ?", "张三")
	if err != nil {
		t.Fatalf("查询更新后数据失败: %v", err)
	}

	if len(queryResult.Data) > 0 {
		age := queryResult.Data[0]["age"]
		if age != int64(30) { // MySQL返回的数字通常是int64
			t.Fatalf("年龄应该是30，得到: %v", age)
		}
	}

	// 删除数据
	deleteSQL := "DELETE FROM test_users WHERE name = ?"
	deleteResult, err := Exec(deleteSQL, "张三")
	if err != nil {
		t.Fatalf("删除数据失败: %v", err)
	}

	if !deleteResult.Success {
		t.Fatalf("删除数据应该成功，得到结果: %+v", deleteResult)
	}

	t.Logf("删除数据成功: %+v", deleteResult)
}

func TestProcedureOperations(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 删除测试表（如果存在）
	DropTable("test_users")

	// 创建测试表
	createSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err = CreateTable(createSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	// 删除存储过程（如果存在）
	DropProcedure("GetUsersByAge")

	// 创建存储过程
	createProcSQL := `
		CREATE PROCEDURE GetUsersByAge(IN min_age INT)
		BEGIN
			SELECT * FROM test_users WHERE age >= min_age;
		END
	`

	createResult, err := CreateProcedure(createProcSQL)
	if err != nil {
		t.Fatalf("创建存储过程失败: %v", err)
	}

	if !createResult.Success {
		t.Fatalf("创建存储过程应该成功，得到结果: %+v", createResult)
	}

	t.Logf("创建存储过程成功: %+v", createResult)

	// 插入测试数据
	ExecWithLastID("INSERT INTO test_users (name, email, age) VALUES (?, ?, ?)", "李四", "lisi@test.com", 28)
	ExecWithLastID("INSERT INTO test_users (name, email, age) VALUES (?, ?, ?)", "王五", "wangwu@test.com", 35)

	// 调用存储过程
	procResult, err := CallProcedure("GetUsersByAge", 30)
	if err != nil {
		t.Fatalf("调用存储过程失败: %v", err)
	}

	if !procResult.Success {
		t.Fatalf("调用存储过程应该成功，得到结果: %+v", procResult)
	}

	t.Logf("调用存储过程成功: %+v", procResult)

	// 检查单结果集的JSON序列化输出
	jsonBytes, _ := json.Marshal(procResult)
	t.Logf("单结果集JSON序列化输出: %s", string(jsonBytes))

	// 验证单结果集使用data字段
	if procResult.Data == nil {
		t.Fatal("单结果集应该使用data字段")
	}
	if procResult.ResultSets != nil {
		t.Fatal("单结果集不应该有result_sets字段")
	}

	// 显示存储过程列表
	showResult, err := ShowProcedures(testConfig.DatabaseName)
	if err != nil {
		t.Fatalf("显示存储过程列表失败: %v", err)
	}

	if !showResult.Success {
		t.Fatalf("显示存储过程列表应该成功，得到结果: %+v", showResult)
	}

	t.Logf("存储过程列表: %+v", showResult)

	// 删除存储过程
	dropResult, err := DropProcedure("GetUsersByAge")
	if err != nil {
		t.Fatalf("删除存储过程失败: %v", err)
	}

	if !dropResult.Success {
		t.Fatalf("删除存储过程应该成功，得到结果: %+v", dropResult)
	}

	t.Logf("删除存储过程成功: %+v", dropResult)
}

func TestTableOperations(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 显示表列表
	showResult, err := ShowTables()
	if err != nil {
		t.Fatalf("显示表列表失败: %v", err)
	}

	if !showResult.Success {
		t.Fatalf("显示表列表应该成功，得到结果: %+v", showResult)
	}

	t.Logf("表列表: %+v", showResult)

	// 删除测试表（如果存在）
	DropTable("test_users")

	// 创建测试表
	createSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE,
			age INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err = CreateTable(createSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}

	// 描述表结构
	descResult, err := DescribeTable("test_users")
	if err != nil {
		t.Fatalf("描述表结构失败: %v", err)
	}

	if !descResult.Success {
		t.Fatalf("描述表结构应该成功，得到结果: %+v", descResult)
	}

	t.Logf("表结构: %+v", descResult)

	// 显示建表语句
	createResult, err := ShowCreateTable("test_users")
	if err != nil {
		t.Fatalf("显示建表语句失败: %v", err)
	}

	if !createResult.Success {
		t.Fatalf("显示建表语句应该成功，得到结果: %+v", createResult)
	}

	t.Logf("建表语句: %+v", createResult)

	// 修改表结构
	alterSQL := "ALTER TABLE test_users ADD COLUMN phone VARCHAR(20)"
	alterResult, err := AlterTable(alterSQL)
	if err != nil {
		t.Fatalf("修改表结构失败: %v", err)
	}

	if !alterResult.Success {
		t.Fatalf("修改表结构应该成功，得到结果: %+v", alterResult)
	}

	t.Logf("修改表结构成功: %+v", alterResult)

	// 验证表结构修改
	descResult2, err := DescribeTable("test_users")
	if err != nil {
		t.Fatalf("描述修改后表结构失败: %v", err)
	}

	// 检查是否有phone字段
	hasPhoneField := false
	for _, row := range descResult2.Data {
		if field, ok := row["Field"]; ok {
			// 字段名可能是字符串或者需要类型转换
			fieldStr := fmt.Sprintf("%v", field)
			if fieldStr == "phone" {
				hasPhoneField = true
				break
			}
		}
	}

	if !hasPhoneField {
		t.Logf("表字段详情: %+v", descResult2)
		t.Fatal("表应该包含phone字段")
	}

	t.Log("表结构修改验证成功")

	// 清理：删除测试表
	dropResult, err := DropTable("test_users")
	if err != nil {
		t.Fatalf("删除表失败: %v", err)
	}

	if !dropResult.Success {
		t.Fatalf("删除表应该成功，得到结果: %+v", dropResult)
	}

	t.Logf("删除表成功: %+v", dropResult)
}

// TestMultipleResultSets 测试多结果集存储过程
func TestMultipleResultSets(t *testing.T) {
	err := InitDB(testConfig)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer CloseDB()

	// 删除测试表（如果存在）
	DropTable("test_users")
	DropTable("test_orders")

	// 创建测试表
	createUsersSQL := `
		CREATE TABLE test_users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			age INT DEFAULT 0
		)
	`
	createOrdersSQL := `
		CREATE TABLE test_orders (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			product VARCHAR(100),
			amount DECIMAL(10,2)
		)
	`

	_, err = CreateTable(createUsersSQL)
	if err != nil {
		t.Fatalf("创建用户表失败: %v", err)
	}

	_, err = CreateTable(createOrdersSQL)
	if err != nil {
		t.Fatalf("创建订单表失败: %v", err)
	}

	// 插入测试数据
	ExecWithLastID("INSERT INTO test_users (name, age) VALUES (?, ?)", "张三", 25)
	ExecWithLastID("INSERT INTO test_users (name, age) VALUES (?, ?)", "李四", 30)
	ExecWithLastID("INSERT INTO test_orders (user_id, product, amount) VALUES (?, ?, ?)", 1, "商品A", 100.50)
	ExecWithLastID("INSERT INTO test_orders (user_id, product, amount) VALUES (?, ?, ?)", 2, "商品B", 200.75)

	// 删除存储过程（如果存在）
	DropProcedure("GetMultipleData")

	// 创建多结果集存储过程
	createProcSQL := `
		CREATE PROCEDURE GetMultipleData()
		BEGIN
			SELECT * FROM test_users;
			SELECT * FROM test_orders;
		END
	`

	createResult, err := CreateProcedure(createProcSQL)
	if err != nil {
		t.Fatalf("创建存储过程失败: %v", err)
	}

	if !createResult.Success {
		t.Fatalf("创建存储过程应该成功，得到结果: %+v", createResult)
	}

	t.Logf("创建存储过程成功: %+v", createResult)

	// 调用存储过程
	procResult, err := CallProcedure("GetMultipleData")
	if err != nil {
		t.Fatalf("调用存储过程失败: %v", err)
	}

	if !procResult.Success {
		t.Fatalf("调用存储过程应该成功，得到结果: %+v", procResult)
	}

	t.Logf("调用存储过程成功: %+v", procResult)

	// 检查JSON序列化输出
	jsonBytes, _ := json.Marshal(procResult)
	t.Logf("JSON序列化输出: %s", string(jsonBytes))

	// 检查是否有多结果集
	if len(procResult.ResultSets) != 2 {
		t.Fatalf("应该有2个结果集，得到: %d", len(procResult.ResultSets))
	}

	// 检查是否有data字段（这里应该没有）
	if procResult.Data != nil {
		t.Logf("警告：多结果集情况下不应该有data字段，但得到了: %+v", procResult.Data)
	}

	t.Logf("第一个结果集（用户）: %+v", procResult.ResultSets[0])
	t.Logf("第二个结果集（订单）: %+v", procResult.ResultSets[1])

	// 清理
	DropProcedure("GetMultipleData")
	DropTable("test_users")
	DropTable("test_orders")
}
