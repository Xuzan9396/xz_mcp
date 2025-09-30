package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"

	"xz_mcp/db/mysql_db"
	"xz_mcp/db/pgsql_db"
	"xz_mcp/db/redis_db"
	"xz_mcp/db/sqlite_db"
)

const (
	ServerName    = "XZ MCP Unified Database Server"
	ServerVersion = "1.0.0"
)

var (
	pgClient    *pgsql_db.PgClient
	redisClient *redis_db.RedisClient
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("%s v%s\n", ServerName, ServerVersion)
		fmt.Println("Integrated: MySQL, PostgreSQL, Redis, SQLite")
		return
	}

	s := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	registerMySQLTools(s)
	registerPostgreSQLTools(s)
	registerRedisTools(s)
	registerSQLiteTools(s)

	log.Printf("Starting %s v%s...\n", ServerName, ServerVersion)
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// registerMySQLTools 注册MySQL相关工具
func registerMySQLTools(s *server.MCPServer) {
	// 1. mysql_connect
	s.AddTool(
		mcp.NewTool("mysql_connect",
			mcp.WithDescription("Connect to MySQL database with dynamic connection parameters"),
			mcp.WithString("username", mcp.Required(), mcp.Description("MySQL username")),
			mcp.WithString("password", mcp.Required(), mcp.Description("MySQL password")),
			mcp.WithString("addr", mcp.Required(), mcp.Description("MySQL server address (host:port)")),
			mcp.WithString("database_name", mcp.Required(), mcp.Description("Database name to connect to")),
			mcp.WithBoolean("debug", mcp.Description("Enable debug mode (default: false)")),
			mcp.WithNumber("max_open_conns", mcp.Description("Maximum number of open connections (default: 100)")),
			mcp.WithNumber("max_idle_conns", mcp.Description("Maximum number of idle connections (default: 50)")),
			mcp.WithNumber("conn_max_lifetime_hours", mcp.Description("Connection maximum lifetime in hours (default: 4)")),
		),
		handleMySQLConnect,
	)

	// 2. mysql_query
	s.AddTool(
		mcp.NewTool("mysql_query",
			mcp.WithDescription("Execute MySQL SELECT query"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL SELECT query to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
		),
		handleMySQLQuery,
	)

	// 3. mysql_exec
	s.AddTool(
		mcp.NewTool("mysql_exec",
			mcp.WithDescription("Execute MySQL INSERT/UPDATE/DELETE operations"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL DML statement to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
		),
		handleMySQLExec,
	)

	// 4. mysql_exec_get_id
	s.AddTool(
		mcp.NewTool("mysql_exec_get_id",
			mcp.WithDescription("Execute MySQL INSERT operation and return the last inserted ID"),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL INSERT statement to execute")),
			mcp.WithArray("args", mcp.Description("Query parameters for prepared statement")),
		),
		handleMySQLExecGetID,
	)

	// 5. mysql_call_procedure
	s.AddTool(
		mcp.NewTool("mysql_call_procedure",
			mcp.WithDescription("Call MySQL stored procedure"),
			mcp.WithString("procedure_name", mcp.Required(), mcp.Description("Name of the stored procedure to call")),
			mcp.WithArray("args", mcp.Description("Arguments to pass to the stored procedure")),
		),
		handleMySQLCallProcedure,
	)

	// 6. mysql_create_procedure
	s.AddTool(
		mcp.NewTool("mysql_create_procedure",
			mcp.WithDescription("Create MySQL stored procedure"),
			mcp.WithString("procedure_sql", mcp.Required(), mcp.Description("Complete CREATE PROCEDURE SQL statement")),
		),
		handleMySQLCreateProcedure,
	)

	// 7. mysql_drop_procedure
	s.AddTool(
		mcp.NewTool("mysql_drop_procedure",
			mcp.WithDescription("Drop MySQL stored procedure"),
			mcp.WithString("procedure_name", mcp.Required(), mcp.Description("Name of the stored procedure to drop")),
		),
		handleMySQLDropProcedure,
	)

	// 8. mysql_show_procedures
	s.AddTool(
		mcp.NewTool("mysql_show_procedures",
			mcp.WithDescription("Show list of stored procedures in the current database"),
			mcp.WithString("database_name", mcp.Description("Database name (if not provided, uses current connection database)")),
		),
		handleMySQLShowProcedures,
	)

	// 9. mysql_create_table
	s.AddTool(
		mcp.NewTool("mysql_create_table",
			mcp.WithDescription("Create MySQL table"),
			mcp.WithString("create_sql", mcp.Required(), mcp.Description("Complete CREATE TABLE SQL statement")),
		),
		handleMySQLCreateTable,
	)

	// 10. mysql_alter_table
	s.AddTool(
		mcp.NewTool("mysql_alter_table",
			mcp.WithDescription("Alter MySQL table structure"),
			mcp.WithString("alter_sql", mcp.Required(), mcp.Description("Complete ALTER TABLE SQL statement")),
		),
		handleMySQLAlterTable,
	)

	// 11. mysql_drop_table
	s.AddTool(
		mcp.NewTool("mysql_drop_table",
			mcp.WithDescription("Drop MySQL table"),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the table to drop")),
		),
		handleMySQLDropTable,
	)

	// 12. mysql_show_tables
	s.AddTool(
		mcp.NewTool("mysql_show_tables",
			mcp.WithDescription("Show list of tables in the current database"),
		),
		handleMySQLShowTables,
	)

	// 13. mysql_describe_table
	s.AddTool(
		mcp.NewTool("mysql_describe_table",
			mcp.WithDescription("Describe MySQL table structure"),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the table to describe")),
		),
		handleMySQLDescribeTable,
	)

	// 14. mysql_show_create_table
	s.AddTool(
		mcp.NewTool("mysql_show_create_table",
			mcp.WithDescription("Show CREATE TABLE statement for a table"),
			mcp.WithString("table_name", mcp.Required(), mcp.Description("Name of the table to show CREATE statement for")),
		),
		handleMySQLShowCreateTable,
	)
}

// handleMySQLConnect MySQL连接处理器
func handleMySQLConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	username, err := request.RequireString("username")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	password, err := request.RequireString("password")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	addr, err := request.RequireString("addr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	databaseName, err := request.RequireString("database_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	debug := request.GetBool("debug", false)
	maxOpenConns := request.GetInt("max_open_conns", 5)
	maxIdleConns := request.GetInt("max_idle_conns", 2)
	connMaxLifetimeHours := request.GetFloat("conn_max_lifetime_hours", 4.0)

	config := mysql_db.ConnectionConfig{
		Username:        username,
		Password:        password,
		Addr:            addr,
		DatabaseName:    databaseName,
		Debug:           debug,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: time.Duration(connMaxLifetimeHours) * time.Hour,
	}

	mysql_db.CloseDB()
	err = mysql_db.InitDB(config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to connect to MySQL: %v", err)), nil
	}

	response := map[string]interface{}{
		"type":    "connection",
		"success": true,
		"message": fmt.Sprintf("Successfully connected to MySQL database '%s' at %s", databaseName, addr),
	}
	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLQuery MySQL查询处理器
func handleMySQLQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []interface{}{}
	if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if argsVal, ok := arguments["args"]; ok {
			if argsSlice, ok := argsVal.([]interface{}); ok {
				args = argsSlice
			}
		}
	}
	result, err := mysql_db.Query(sql, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLExec MySQL执行处理器
func handleMySQLExec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []interface{}{}
	if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if argsVal, ok := arguments["args"]; ok {
			if argsSlice, ok := argsVal.([]interface{}); ok {
				args = argsSlice
			}
		}
	}
	result, err := mysql_db.Exec(sql, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Execution failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLExecGetID MySQL执行并获取ID处理器
func handleMySQLExecGetID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	sql, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []interface{}{}
	if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if argsVal, ok := arguments["args"]; ok {
			if argsSlice, ok := argsVal.([]interface{}); ok {
				args = argsSlice
			}
		}
	}
	result, err := mysql_db.ExecWithLastID(sql, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Execution failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLCallProcedure 调用存储过程处理器
func handleMySQLCallProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	procName, err := request.RequireString("procedure_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []interface{}{}
	if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if argsVal, ok := arguments["args"]; ok {
			if argsSlice, ok := argsVal.([]interface{}); ok {
				args = argsSlice
			}
		}
	}
	result, err := mysql_db.CallProcedure(procName, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Procedure call failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLCreateProcedure 创建存储过程处理器
func handleMySQLCreateProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	procedureSQL, err := request.RequireString("procedure_sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.CreateProcedure(procedureSQL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Create procedure failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLDropProcedure 删除存储过程处理器
func handleMySQLDropProcedure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	procName, err := request.RequireString("procedure_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.DropProcedure(procName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Drop procedure failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLShowProcedures 显示存储过程列表处理器
func handleMySQLShowProcedures(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	databaseName := request.GetString("database_name", "")
	if databaseName == "" {
		return mcp.NewToolResultError("database_name parameter is required"), nil
	}
	result, err := mysql_db.ShowProcedures(databaseName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Show procedures failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLCreateTable 创建表处理器
func handleMySQLCreateTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	createSQL, err := request.RequireString("create_sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.CreateTable(createSQL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Create table failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLAlterTable 修改表结构处理器
func handleMySQLAlterTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	alterSQL, err := request.RequireString("alter_sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.AlterTable(alterSQL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Alter table failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLDropTable 删除表处理器
func handleMySQLDropTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.DropTable(tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Drop table failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLShowTables 显示表列表处理器
func handleMySQLShowTables(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	result, err := mysql_db.ShowTables()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Show tables failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLDescribeTable 描述表结构处理器
func handleMySQLDescribeTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.DescribeTable(tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Describe table failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMySQLShowCreateTable 显示建表语句处理器
func handleMySQLShowCreateTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !mysql_db.IsConnected() {
		return mcp.NewToolResultError("Database not connected. Use mysql_connect first"), nil
	}
	tableName, err := request.RequireString("table_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	result, err := mysql_db.ShowCreateTable(tableName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Show create table failed: %v", err)), nil
	}
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// registerPostgreSQLTools 注册PostgreSQL相关工具
func registerPostgreSQLTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("pgsql_connect",
			mcp.WithDescription("连接到PostgreSQL服务器"),
			mcp.WithString("host", mcp.Required()),
			mcp.WithNumber("port", mcp.DefaultNumber(5432)),
			mcp.WithString("user", mcp.Required()),
			mcp.WithString("password", mcp.Required()),
			mcp.WithString("database", mcp.Required()),
			mcp.WithString("sslmode", mcp.DefaultString("disable")),
		),
		handlePgConnect,
	)

	s.AddTool(
		mcp.NewTool("pgsql_query",
			mcp.WithDescription("执行PostgreSQL SELECT查询"),
			mcp.WithString("sql", mcp.Required()),
		),
		handlePgQuery,
	)

	s.AddTool(
		mcp.NewTool("pgsql_exec",
			mcp.WithDescription("执行PostgreSQL INSERT/UPDATE/DELETE操作"),
			mcp.WithString("sql", mcp.Required()),
		),
		handlePgExec,
	)
}

// PostgreSQL辅助函数
func getStringParam(args map[string]interface{}, key string, defaultValue string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultValue
}

func getNumberParam(args map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := args[key].(float64); ok {
		return val
	}
	return defaultValue
}

// handlePgConnect PostgreSQL连接处理器
func handlePgConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments == nil {
		return nil, fmt.Errorf("缺少必需参数")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("参数格式错误")
	}

	config := pgsql_db.PgConfig{
		Host:     getStringParam(args, "host", "localhost"),
		Port:     int(getNumberParam(args, "port", 5432)),
		User:     getStringParam(args, "user", ""),
		Password: getStringParam(args, "password", ""),
		Database: getStringParam(args, "database", ""),
		SSLMode:  getStringParam(args, "sslmode", "disable"),
	}

	client, err := pgsql_db.NewPgClient(config)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL连接失败: %v", err)
	}
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("PostgreSQL连接测试失败: %v", err)
	}

	if pgClient != nil {
		pgClient.Close()
	}
	pgClient = client

	result := map[string]interface{}{
		"status":  "success",
		"message": "PostgreSQL连接成功",
		"config": map[string]interface{}{
			"host":     config.Host,
			"port":     config.Port,
			"database": config.Database,
			"user":     config.User,
			"sslmode":  config.SSLMode,
		},
	}
	resultBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultBytes)), nil
}

// handlePgQuery PostgreSQL查询处理器
func handlePgQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if pgClient == nil {
		return nil, fmt.Errorf("请先连接到PostgreSQL服务器")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("参数格式错误")
	}
	sql := getStringParam(args, "sql", "")
	if sql == "" {
		return nil, fmt.Errorf("SQL语句不能为空")
	}
	result, err := pgClient.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	resultBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultBytes)), nil
}

// handlePgExec PostgreSQL执行处理器
func handlePgExec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if pgClient == nil {
		return nil, fmt.Errorf("请先连接到PostgreSQL服务器")
	}
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("参数格式错误")
	}
	sql := getStringParam(args, "sql", "")
	if sql == "" {
		return nil, fmt.Errorf("SQL语句不能为空")
	}
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))
	var result interface{}
	var err error
	if strings.HasPrefix(upperSQL, "INSERT") && strings.Contains(upperSQL, "RETURNING") {
		result, err = pgClient.ExecWithLastInsertId(ctx, sql)
	} else {
		result, err = pgClient.Exec(ctx, sql)
	}
	if err != nil {
		return nil, fmt.Errorf("执行失败: %v", err)
	}
	resultBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultBytes)), nil
}

// registerRedisTools 注册Redis相关工具
func registerRedisTools(s *server.MCPServer) {
	// 1. redis_connect
	s.AddTool(
		mcp.NewTool("redis_connect",
			mcp.WithDescription("连接到Redis服务器"),
			mcp.WithString("addr", mcp.Required(), mcp.Description("Redis服务器地址 (例如: 127.0.0.1:6379)")),
			mcp.WithString("password", mcp.Description("Redis密码")),
			mcp.WithNumber("db", mcp.DefaultNumber(0), mcp.Description("Redis数据库编号")),
			mcp.WithBoolean("ssl_insecure_skip_verify", mcp.Description("是否跳过SSL证书验证，设置为true时启用跳过验证(默认不设置)")),
		),
		handleRedisConnect,
	)

	// 2. redis_disconnect
	s.AddTool(
		mcp.NewTool("redis_disconnect",
			mcp.WithDescription("断开Redis连接"),
		),
		handleRedisDisconnect,
	)

	// 3. redis_ping
	s.AddTool(
		mcp.NewTool("redis_ping",
			mcp.WithDescription("测试Redis连接"),
		),
		handleRedisPing,
	)

	// 4. redis_command
	s.AddTool(
		mcp.NewTool("redis_command",
			mcp.WithDescription("执行Redis命令"),
			mcp.WithString("command", mcp.Required(), mcp.Description("Redis命令 (例如: SET key value 或 GET key)")),
		),
		handleRedisCommand,
	)

	// 5. redis_lua
	s.AddTool(
		mcp.NewTool("redis_lua",
			mcp.WithDescription("执行Lua脚本"),
			mcp.WithString("script", mcp.Required(), mcp.Description("Lua脚本代码")),
			mcp.WithArray("keys", mcp.Description("脚本中使用的键名列表")),
			mcp.WithArray("args", mcp.Description("脚本参数列表")),
		),
		handleRedisLua,
	)

	// 6. redis_info
	s.AddTool(
		mcp.NewTool("redis_info",
			mcp.WithDescription("获取Redis服务器信息"),
			mcp.WithString("section", mcp.Description("信息部分 (例如: server, memory, replication)")),
		),
		handleRedisInfo,
	)

	// 7. redis_keys
	s.AddTool(
		mcp.NewTool("redis_keys",
			mcp.WithDescription("获取匹配模式的键"),
			mcp.WithString("pattern", mcp.DefaultString("*"), mcp.Description("键名模式 (例如: user:*, cache:*)")),
		),
		handleRedisKeys,
	)

	// 8. redis_key_info
	s.AddTool(
		mcp.NewTool("redis_key_info",
			mcp.WithDescription("获取键的详细信息"),
			mcp.WithString("key", mcp.Required(), mcp.Description("键名")),
		),
		handleRedisKeyInfo,
	)

	// 9. redis_del
	s.AddTool(
		mcp.NewTool("redis_del",
			mcp.WithDescription("删除一个或多个键"),
			mcp.WithArray("keys", mcp.Required(), mcp.Description("要删除的键名列表")),
		),
		handleRedisDel,
	)

	// 10. redis_expire
	s.AddTool(
		mcp.NewTool("redis_expire",
			mcp.WithDescription("设置键的过期时间"),
			mcp.WithString("key", mcp.Required(), mcp.Description("键名")),
			mcp.WithNumber("seconds", mcp.Required(), mcp.Description("过期时间(秒)")),
		),
		handleRedisExpire,
	)

	// 11. redis_string
	s.AddTool(
		mcp.NewTool("redis_string",
			mcp.WithDescription("字符串操作 (SET, GET, INCR, DECR等)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("SET", "GET", "MGET", "MSET", "INCR", "DECR", "INCRBY", "DECRBY"), mcp.Description("操作类型")),
			mcp.WithString("key", mcp.Description("键名")),
			mcp.WithString("value", mcp.Description("值")),
			mcp.WithArray("keys", mcp.Description("键名列表 (用于MGET/MSET)")),
			mcp.WithArray("values", mcp.Description("值列表 (用于MSET)")),
			mcp.WithNumber("increment", mcp.Description("增量值 (用于INCRBY/DECRBY)")),
			mcp.WithNumber("expire", mcp.Description("过期时间(秒, 用于SET)")),
		),
		handleRedisString,
	)

	// 12. redis_hash
	s.AddTool(
		mcp.NewTool("redis_hash",
			mcp.WithDescription("哈希操作 (HSET, HGET, HGETALL, HDEL等)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("HSET", "HGET", "HGETALL", "HDEL", "HEXISTS", "HKEYS", "HLEN"), mcp.Description("操作类型")),
			mcp.WithString("key", mcp.Required(), mcp.Description("哈希键名")),
			mcp.WithString("field", mcp.Description("字段名")),
			mcp.WithString("value", mcp.Description("字段值")),
			mcp.WithArray("fields", mcp.Description("字段名列表")),
			mcp.WithArray("values", mcp.Description("字段值列表")),
		),
		handleRedisHash,
	)

	// 13. redis_list
	s.AddTool(
		mcp.NewTool("redis_list",
			mcp.WithDescription("列表操作 (LPUSH, RPUSH, LPOP, RPOP, LRANGE等)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("LPUSH", "RPUSH", "LPOP", "RPOP", "LRANGE", "LLEN"), mcp.Description("操作类型")),
			mcp.WithString("key", mcp.Required(), mcp.Description("列表键名")),
			mcp.WithArray("values", mcp.Description("要添加的值列表")),
			mcp.WithNumber("start", mcp.Description("起始位置 (用于LRANGE)")),
			mcp.WithNumber("stop", mcp.Description("结束位置 (用于LRANGE)")),
		),
		handleRedisList,
	)

	// 14. redis_set
	s.AddTool(
		mcp.NewTool("redis_set",
			mcp.WithDescription("集合操作 (SADD, SMEMBERS, SREM, SISMEMBER等)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("SADD", "SMEMBERS", "SREM", "SISMEMBER", "SCARD"), mcp.Description("操作类型")),
			mcp.WithString("key", mcp.Required(), mcp.Description("集合键名")),
			mcp.WithArray("members", mcp.Description("集合成员列表")),
			mcp.WithString("member", mcp.Description("集合成员")),
		),
		handleRedisSet,
	)

	// 15. redis_zset
	s.AddTool(
		mcp.NewTool("redis_zset",
			mcp.WithDescription("有序集合操作 (ZADD, ZRANGE, ZREM, ZSCORE等)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("ZADD", "ZRANGE", "ZREM", "ZSCORE", "ZCARD"), mcp.Description("操作类型")),
			mcp.WithString("key", mcp.Required(), mcp.Description("有序集合键名")),
			mcp.WithArray("members", mcp.Description("成员列表")),
			mcp.WithArray("scores", mcp.Description("分数列表")),
			mcp.WithString("member", mcp.Description("成员名")),
			mcp.WithNumber("start", mcp.Description("起始位置 (用于ZRANGE)")),
			mcp.WithNumber("stop", mcp.Description("结束位置 (用于ZRANGE)")),
		),
		handleRedisZSet,
	)

	// 16. redis_db
	s.AddTool(
		mcp.NewTool("redis_db",
			mcp.WithDescription("数据库管理操作 (DBSIZE, FLUSHDB, FLUSHALL)"),
			mcp.WithString("operation", mcp.Required(), mcp.Enum("DBSIZE", "FLUSHDB", "FLUSHALL"), mcp.Description("操作类型")),
		),
		handleRedisDB,
	)
}

// Redis连接处理器
func handleRedisConnect(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	addr, err := req.RequireString("addr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	password := req.GetString("password", "")
	db := req.GetInt("db", 0)

	config := redis_db.RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	}

	// 处理 ssl_insecure_skip_verify 参数
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if sslSkipVerify, exists := rawArgs["ssl_insecure_skip_verify"]; exists {
			if skipVerify, ok := sslSkipVerify.(bool); ok {
				config.SSLInsecureSkipVerify = &skipVerify
			}
		}
	}

	// 如果已有连接，先关闭
	if redisClient != nil {
		redisClient.Close()
	}

	// 创建新连接
	redisClient = redis_db.NewRedisClient(config)

	// 测试连接
	if err := redisClient.Ping(ctx); err != nil {
		redisClient.Close()
		redisClient = nil
		return mcp.NewToolResultError(fmt.Sprintf("连接失败: %v", err)), nil
	}

	result := map[string]interface{}{
		"status": "connected",
		"addr":   addr,
		"db":     db,
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Redis断开连接处理器
func handleRedisDisconnect(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接"), nil
	}

	if err := redisClient.Close(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("断开连接失败: %v", err)), nil
	}

	redisClient = nil
	return mcp.NewToolResultText("{\"status\": \"disconnected\"}"), nil
}

// Redis连接测试处理器
func handleRedisPing(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接"), nil
	}

	if err := redisClient.Ping(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("连接测试失败: %v", err)), nil
	}

	return mcp.NewToolResultText("{\"status\": \"PONG\"}"), nil
}

// Redis命令执行处理器
func handleRedisCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args, err := redis_db.ParseRedisCommand(command)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("解析命令失败: %v", err)), nil
	}

	result, err := redisClient.ExecuteCommand(ctx, args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行命令失败: %v", err)), nil
	}

	formattedResult, err := redis_db.FormatRedisResult(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("格式化结果失败: %v", err)), nil
	}

	return mcp.NewToolResultText(formattedResult), nil
}

// Redis Lua脚本执行处理器
func handleRedisLua(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	script, err := req.RequireString("script")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 解析keys参数
	var keys []string
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if keysArray, ok := rawArgs["keys"].([]interface{}); ok {
			for _, key := range keysArray {
				if keyStr, ok := key.(string); ok {
					keys = append(keys, keyStr)
				}
			}
		}
	}

	// 解析args参数
	var args []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if argsArray, ok := rawArgs["args"].([]interface{}); ok {
			args = argsArray
		}
	}

	result, err := redisClient.ExecuteLuaScript(ctx, script, keys, args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("执行Lua脚本失败: %v", err)), nil
	}

	formattedResult, err := redis_db.FormatRedisResult(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("格式化结果失败: %v", err)), nil
	}

	return mcp.NewToolResultText(formattedResult), nil
}

// Redis信息查询处理器
func handleRedisInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	section := req.GetString("section", "")
	info, err := redisClient.GetInfo(ctx, section)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取信息失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("\"%s\"", info)), nil
}

// Redis键查询处理器
func handleRedisKeys(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	pattern := req.GetString("pattern", "*")
	keys, err := redisClient.GetKeys(ctx, pattern)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取键列表失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(keys)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Redis键信息查询处理器
func handleRedisKeyInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 检查键是否存在
	exists, err := redisClient.Exists(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("检查键存在性失败: %v", err)), nil
	}

	if exists == 0 {
		return mcp.NewToolResultText("{\"exists\": false}"), nil
	}

	// 获取键类型
	keyType, err := redisClient.GetType(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取键类型失败: %v", err)), nil
	}

	// 获取TTL
	ttl, err := redisClient.GetTTL(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取TTL失败: %v", err)), nil
	}

	result := map[string]interface{}{
		"exists": true,
		"type":   keyType,
		"ttl":    ttl.Seconds(),
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Redis键删除处理器
func handleRedisDel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	var keys []string
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if keysArray, ok := rawArgs["keys"].([]interface{}); ok {
			for _, key := range keysArray {
				if keyStr, ok := key.(string); ok {
					keys = append(keys, keyStr)
				}
			}
		}
	}

	if len(keys) == 0 {
		return mcp.NewToolResultError("缺少keys参数"), nil
	}

	if len(keys) == 0 {
		return mcp.NewToolResultError("keys列表为空"), nil
	}

	deleted, err := redisClient.Delete(ctx, keys...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("删除键失败: %v", err)), nil
	}

	result := map[string]interface{}{
		"deleted": deleted,
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// Redis过期时间设置处理器
func handleRedisExpire(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	seconds, err := req.RequireFloat("seconds")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	expiration := time.Duration(seconds) * time.Second
	err = redisClient.SetExpire(ctx, key, expiration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("设置过期时间失败: %v", err)), nil
	}

	return mcp.NewToolResultText("{\"status\": \"ok\"}"), nil
}

// Redis字符串操作处理器
func handleRedisString(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "SET":
		return handleStringSet(ctx, req)
	case "GET":
		return handleStringGet(ctx, req)
	case "MGET":
		return handleStringMGet(ctx, req)
	case "MSET":
		return handleStringMSet(ctx, req)
	case "INCR":
		return handleStringIncr(ctx, req)
	case "DECR":
		return handleStringDecr(ctx, req)
	case "INCRBY":
		return handleStringIncrBy(ctx, req)
	case "DECRBY":
		return handleStringDecrBy(ctx, req)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// 字符串SET操作
func handleStringSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := req.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	expire := req.GetInt("expire", 0)
	expiration := time.Duration(expire) * time.Second

	err = redisClient.Set(ctx, key, value, expiration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SET操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText("{\"status\": \"OK\"}"), nil
}

// 字符串GET操作
func handleStringGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := redisClient.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return mcp.NewToolResultText("null"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("GET操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("\"%s\"", value)), nil
}

// 字符串MGET操作
func handleStringMGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var keys []string
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if keysArray, ok := rawArgs["keys"].([]interface{}); ok {
			for _, key := range keysArray {
				if keyStr, ok := key.(string); ok {
					keys = append(keys, keyStr)
				}
			}
		}
	}

	if len(keys) == 0 {
		return mcp.NewToolResultError("缺少keys参数"), nil
	}

	if len(keys) == 0 {
		return mcp.NewToolResultError("keys列表为空"), nil
	}

	values, err := redisClient.MGet(ctx, keys...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("MGET操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(values)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 字符串MSET操作
func handleStringMSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var keysArray, valuesArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if keys, ok := rawArgs["keys"].([]interface{}); ok {
			keysArray = keys
		}
		if values, ok := rawArgs["values"].([]interface{}); ok {
			valuesArray = values
		}
	}

	if len(keysArray) == 0 || len(valuesArray) == 0 {
		return mcp.NewToolResultError("缺少keys或values参数"), nil
	}

	if len(keysArray) != len(valuesArray) {
		return mcp.NewToolResultError("keys和values数量不匹配"), nil
	}

	var pairs []interface{}
	for i := 0; i < len(keysArray); i++ {
		pairs = append(pairs, keysArray[i], valuesArray[i])
	}

	err := redisClient.MSet(ctx, pairs...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("MSET操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText("{\"status\": \"OK\"}"), nil
}

// 字符串INCR操作
func handleStringIncr(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := redisClient.Incr(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("INCR操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 字符串DECR操作
func handleStringDecr(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := redisClient.Decr(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DECR操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 字符串INCRBY操作
func handleStringIncrBy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	increment, err := req.RequireFloat("increment")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := redisClient.IncrBy(ctx, key, int64(increment))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("INCRBY操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 字符串DECRBY操作
func handleStringDecrBy(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	decrement, err := req.RequireFloat("increment")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := redisClient.DecrBy(ctx, key, int64(decrement))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("DECRBY操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// Redis哈希操作处理器
func handleRedisHash(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "HSET":
		return handleHashSet(ctx, req, key)
	case "HGET":
		return handleHashGet(ctx, req, key)
	case "HGETALL":
		return handleHashGetAll(ctx, req, key)
	case "HDEL":
		return handleHashDel(ctx, req, key)
	case "HEXISTS":
		return handleHashExists(ctx, req, key)
	case "HKEYS":
		return handleHashKeys(ctx, req, key)
	case "HLEN":
		return handleHashLen(ctx, req, key)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// 哈希HSET操作
func handleHashSet(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var fieldsArray, valuesArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if fields, ok := rawArgs["fields"].([]interface{}); ok {
			fieldsArray = fields
		}
		if values, ok := rawArgs["values"].([]interface{}); ok {
			valuesArray = values
		}
	}

	if len(fieldsArray) > 0 && len(valuesArray) > 0 {
		if len(fieldsArray) != len(valuesArray) {
			return mcp.NewToolResultError("fields和values数量不匹配"), nil
		}

		var pairs []interface{}
		for i := 0; i < len(fieldsArray); i++ {
			pairs = append(pairs, fieldsArray[i], valuesArray[i])
		}

		result, err := redisClient.HSet(ctx, key, pairs...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("HSET操作失败: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
	}

	// 单个字段设置
	field, err := req.RequireString("field")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := req.RequireString("value")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := redisClient.HSet(ctx, key, field, value)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HSET操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 哈希HGET操作
func handleHashGet(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	field, err := req.RequireString("field")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	value, err := redisClient.HGet(ctx, key, field)
	if err != nil {
		if err == redis.Nil {
			return mcp.NewToolResultText("null"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("HGET操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("\"%s\"", value)), nil
}

// 哈希HGETALL操作
func handleHashGetAll(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	result, err := redisClient.HGetAll(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HGETALL操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 哈希HDEL操作
func handleHashDel(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var fieldsArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if fields, ok := rawArgs["fields"].([]interface{}); ok {
			fieldsArray = fields
		}
	}

	if len(fieldsArray) == 0 {
		field, err := req.RequireString("field")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fieldsArray = []interface{}{field}
	}

	var fields []string
	for _, field := range fieldsArray {
		if fieldStr, ok := field.(string); ok {
			fields = append(fields, fieldStr)
		}
	}

	result, err := redisClient.HDel(ctx, key, fields...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HDEL操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 哈希HEXISTS操作
func handleHashExists(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	field, err := req.RequireString("field")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	exists, err := redisClient.HExists(ctx, key, field)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HEXISTS操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%t", exists)), nil
}

// 哈希HKEYS操作
func handleHashKeys(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	keys, err := redisClient.HKeys(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HKEYS操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(keys)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 哈希HLEN操作
func handleHashLen(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	length, err := redisClient.HLen(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("HLEN操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", length)), nil
}

// Redis列表操作处理器
func handleRedisList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "LPUSH":
		return handleListLPush(ctx, req, key)
	case "RPUSH":
		return handleListRPush(ctx, req, key)
	case "LPOP":
		return handleListLPop(ctx, req, key)
	case "RPOP":
		return handleListRPop(ctx, req, key)
	case "LRANGE":
		return handleListLRange(ctx, req, key)
	case "LLEN":
		return handleListLLen(ctx, req, key)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// 列表LPUSH操作
func handleListLPush(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var valuesArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if values, ok := rawArgs["values"].([]interface{}); ok {
			valuesArray = values
		}
	}

	if len(valuesArray) == 0 {
		return mcp.NewToolResultError("缺少values参数"), nil
	}

	result, err := redisClient.LPush(ctx, key, valuesArray...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LPUSH操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 列表RPUSH操作
func handleListRPush(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var valuesArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if values, ok := rawArgs["values"].([]interface{}); ok {
			valuesArray = values
		}
	}

	if len(valuesArray) == 0 {
		return mcp.NewToolResultError("缺少values参数"), nil
	}

	result, err := redisClient.RPush(ctx, key, valuesArray...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("RPUSH操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 列表LPOP操作
func handleListLPop(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	result, err := redisClient.LPop(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return mcp.NewToolResultText("null"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("LPOP操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("\"%s\"", result)), nil
}

// 列表RPOP操作
func handleListRPop(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	result, err := redisClient.RPop(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return mcp.NewToolResultText("null"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("RPOP操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("\"%s\"", result)), nil
}

// 列表LRANGE操作
func handleListLRange(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	start := req.GetInt("start", 0)
	stop := req.GetInt("stop", -1)

	result, err := redisClient.LRange(ctx, key, int64(start), int64(stop))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LRANGE操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 列表LLEN操作
func handleListLLen(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	length, err := redisClient.LLen(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("LLEN操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", length)), nil
}

// Redis集合操作处理器
func handleRedisSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "SADD":
		return handleSetSAdd(ctx, req, key)
	case "SMEMBERS":
		return handleSetSMembers(ctx, req, key)
	case "SREM":
		return handleSetSRem(ctx, req, key)
	case "SISMEMBER":
		return handleSetSIsMember(ctx, req, key)
	case "SCARD":
		return handleSetSCard(ctx, req, key)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// 集合SADD操作
func handleSetSAdd(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var membersArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if members, ok := rawArgs["members"].([]interface{}); ok {
			membersArray = members
		}
	}

	if len(membersArray) == 0 {
		return mcp.NewToolResultError("缺少members参数"), nil
	}

	result, err := redisClient.SAdd(ctx, key, membersArray...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SADD操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 集合SMEMBERS操作
func handleSetSMembers(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	members, err := redisClient.SMembers(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SMEMBERS操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(members)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 集合SREM操作
func handleSetSRem(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var membersArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if members, ok := rawArgs["members"].([]interface{}); ok {
			membersArray = members
		}
	}

	if len(membersArray) == 0 {
		return mcp.NewToolResultError("缺少members参数"), nil
	}

	result, err := redisClient.SRem(ctx, key, membersArray...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SREM操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 集合SISMEMBER操作
func handleSetSIsMember(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	member, err := req.RequireString("member")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	exists, err := redisClient.SIsMember(ctx, key, member)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SISMEMBER操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%t", exists)), nil
}

// 集合SCARD操作
func handleSetSCard(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	count, err := redisClient.SCard(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SCARD操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", count)), nil
}

// Redis有序集合操作处理器
func handleRedisZSet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	key, err := req.RequireString("key")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "ZADD":
		return handleZSetZAdd(ctx, req, key)
	case "ZRANGE":
		return handleZSetZRange(ctx, req, key)
	case "ZREM":
		return handleZSetZRem(ctx, req, key)
	case "ZSCORE":
		return handleZSetZScore(ctx, req, key)
	case "ZCARD":
		return handleZSetZCard(ctx, req, key)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// 有序集合ZADD操作
func handleZSetZAdd(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var membersArray, scoresArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if members, ok := rawArgs["members"].([]interface{}); ok {
			membersArray = members
		}
		if scores, ok := rawArgs["scores"].([]interface{}); ok {
			scoresArray = scores
		}
	}

	if len(membersArray) == 0 || len(scoresArray) == 0 {
		return mcp.NewToolResultError("缺少members或scores参数"), nil
	}

	if len(membersArray) != len(scoresArray) {
		return mcp.NewToolResultError("members和scores数量不匹配"), nil
	}

	var members []redis.Z
	for i := 0; i < len(membersArray); i++ {
		var score float64
		switch s := scoresArray[i].(type) {
		case float64:
			score = s
		case int:
			score = float64(s)
		case string:
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				score = f
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("无效的分数: %s", s)), nil
			}
		default:
			return mcp.NewToolResultError(fmt.Sprintf("无效的分数类型: %T", s)), nil
		}

		members = append(members, redis.Z{
			Score:  score,
			Member: membersArray[i],
		})
	}

	result, err := redisClient.ZAdd(ctx, key, members...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ZADD操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 有序集合ZRANGE操作
func handleZSetZRange(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	start := req.GetInt("start", 0)
	stop := req.GetInt("stop", -1)

	result, err := redisClient.ZRange(ctx, key, int64(start), int64(stop))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ZRANGE操作失败: %v", err)), nil
	}

	jsonResult, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// 有序集合ZREM操作
func handleZSetZRem(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	var membersArray []interface{}
	if rawArgs := req.GetArguments(); rawArgs != nil {
		if members, ok := rawArgs["members"].([]interface{}); ok {
			membersArray = members
		}
	}

	if len(membersArray) == 0 {
		return mcp.NewToolResultError("缺少members参数"), nil
	}

	result, err := redisClient.ZRem(ctx, key, membersArray...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ZREM操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", result)), nil
}

// 有序集合ZSCORE操作
func handleZSetZScore(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	member, err := req.RequireString("member")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	score, err := redisClient.ZScore(ctx, key, member)
	if err != nil {
		if err == redis.Nil {
			return mcp.NewToolResultText("null"), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("ZSCORE操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%f", score)), nil
}

// 有序集合ZCARD操作
func handleZSetZCard(ctx context.Context, req mcp.CallToolRequest, key string) (*mcp.CallToolResult, error) {
	count, err := redisClient.ZCard(ctx, key)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ZCARD操作失败: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", count)), nil
}

// Redis数据库管理处理器
func handleRedisDB(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if redisClient == nil {
		return mcp.NewToolResultError("没有活动的Redis连接，请先执行 redis_connect"), nil
	}

	operation, err := req.RequireString("operation")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch strings.ToUpper(operation) {
	case "DBSIZE":
		size, err := redisClient.GetDBSize(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("DBSIZE操作失败: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("%d", size)), nil

	case "FLUSHDB":
		err := redisClient.FlushDB(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("FLUSHDB操作失败: %v", err)), nil
		}
		return mcp.NewToolResultText("{\"status\": \"OK\"}"), nil

	case "FLUSHALL":
		err := redisClient.FlushAll(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("FLUSHALL操作失败: %v", err)), nil
		}
		return mcp.NewToolResultText("{\"status\": \"OK\"}"), nil

	default:
		return mcp.NewToolResultError(fmt.Sprintf("不支持的操作: %s", operation)), nil
	}
}

// registerSQLiteTools 注册SQLite相关工具
func registerSQLiteTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("sqlite_query",
			mcp.WithDescription("Execute SQL query on SQLite database"),
			mcp.WithString("db_path", mcp.Required(), mcp.Description("Path to the SQLite database file")),
			mcp.WithString("sql", mcp.Required(), mcp.Description("SQL query to execute")),
		),
		handleSQLiteQuery,
	)
}

// handleSQLiteQuery SQLite查询处理器(支持SELECT和DML)
func handleSQLiteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbPath, err := request.RequireString("db_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sqlQuery, err := request.RequireString("sql")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	err = sqlite_db.InitDB(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to connect to database: %v", err)), nil
	}
	defer sqlite_db.CloseDB()

	sqlTrimmed := strings.TrimSpace(strings.ToUpper(sqlQuery))
	isModification := strings.HasPrefix(sqlTrimmed, "INSERT") ||
		strings.HasPrefix(sqlTrimmed, "UPDATE") ||
		strings.HasPrefix(sqlTrimmed, "DELETE")

	if isModification {
		result, err := sqlite_db.Db().Exec(sqlQuery)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get rows affected: %v", err)), nil
		}
		response := map[string]interface{}{
			"type":         "modification",
			"rowsAffected": rowsAffected,
		}
		if strings.HasPrefix(sqlTrimmed, "INSERT") {
			lastId, err := result.LastInsertId()
			if err == nil {
				response["lastInsertId"] = lastId
			}
		}
		jsonData, _ := json.MarshalIndent(response, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	}

	// Handle SELECT queries
	rows, err := sqlite_db.Db().Query(sqlQuery)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get columns: %v", err)), nil
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scan row: %v", err)), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("Row iteration error: %v", err)), nil
	}
	response := map[string]interface{}{
		"type":  "select",
		"data":  results,
		"count": len(results),
	}
	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}
