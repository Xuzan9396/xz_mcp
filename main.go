package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"xz_mcp/db/mysql_db"
	"xz_mcp/db/pgsql_db"
	"xz_mcp/db/redis_db"
	"xz_mcp/db/sqlite_db"
)

const (
	ServerName = "XZ MCP Database Server"
)

var (
	ServerVersion = "dev" // 将在编译时通过 ldflags 注入实际版本
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
	// 1. redis_connect - 连接到Redis服务器
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

	// 2. redis_command - 执行任意Redis命令
	s.AddTool(
		mcp.NewTool("redis_command",
			mcp.WithDescription("执行任意Redis命令"),
			mcp.WithString("command", mcp.Required(), mcp.Description("Redis命令 (例如: SET key value 或 GET key)")),
		),
		handleRedisCommand,
	)

	// 3. redis_lua - 执行Lua脚本
	s.AddTool(
		mcp.NewTool("redis_lua",
			mcp.WithDescription("执行Lua脚本"),
			mcp.WithString("script", mcp.Required(), mcp.Description("Lua脚本代码")),
			mcp.WithArray("keys", mcp.Description("脚本中使用的键名列表")),
			mcp.WithArray("args", mcp.Description("脚本参数列表")),
		),
		handleRedisLua,
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
