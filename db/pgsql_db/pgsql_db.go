package pgsql_db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PgConfig PostgreSQL配置结构
type PgConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}

// PgClient PostgreSQL客户端包装器
type PgClient struct {
	db     *sqlx.DB
	config PgConfig
}

// QueryResult 查询结果结构
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int                      `json:"count"`
}

// ExecResult 执行结果结构
type ExecResult struct {
	RowsAffected int64  `json:"rows_affected"`
	LastInsertId int64  `json:"last_insert_id,omitempty"`
	Message      string `json:"message"`
}

// NewPgClient 创建新的PostgreSQL客户端
func NewPgClient(config PgConfig) (*PgClient, error) {
	// 设置默认值
	if config.Port == 0 {
		config.Port = 5432
	}
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(4 * time.Hour)

	return &PgClient{
		db:     db,
		config: config,
	}, nil
}

// Close 关闭PostgreSQL连接
func (p *PgClient) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Ping 测试PostgreSQL连接
func (p *PgClient) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// GetInfo 获取PostgreSQL服务器信息
func (p *PgClient) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// 获取版本信息
	var version string
	err := p.db.GetContext(ctx, &version, "SELECT version()")
	if err != nil {
		return nil, fmt.Errorf("获取版本信息失败: %w", err)
	}
	info["version"] = version

	// 获取当前数据库
	var database string
	err = p.db.GetContext(ctx, &database, "SELECT current_database()")
	if err != nil {
		return nil, fmt.Errorf("获取当前数据库失败: %w", err)
	}
	info["current_database"] = database

	// 获取当前用户
	var user string
	err = p.db.GetContext(ctx, &user, "SELECT current_user")
	if err != nil {
		return nil, fmt.Errorf("获取当前用户失败: %w", err)
	}
	info["current_user"] = user

	// 获取连接信息
	info["config"] = map[string]interface{}{
		"host":     p.config.Host,
		"port":     p.config.Port,
		"database": p.config.Database,
		"user":     p.config.User,
		"sslmode":  p.config.SSLMode,
	}

	return info, nil
}

// Query 执行SELECT查询
func (p *PgClient) Query(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询执行失败: %w", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	// 准备存储结果
	var result []map[string]interface{}

	for rows.Next() {
		// 创建接收器
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		// 扫描行数据
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 构建行数据
		row := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			if val == nil {
				row[colName] = nil
			} else {
				switch v := val.(type) {
				case []byte:
					row[colName] = string(v)
				default:
					row[colName] = v
				}
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("行遍历错误: %w", err)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    result,
		Count:   len(result),
	}, nil
}

// Exec 执行INSERT/UPDATE/DELETE等语句
func (p *PgClient) Exec(ctx context.Context, query string, args ...interface{}) (*ExecResult, error) {
	result, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("执行SQL失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("获取影响行数失败: %w", err)
	}

	execResult := &ExecResult{
		RowsAffected: rowsAffected,
		Message:      "执行成功",
	}

	return execResult, nil
}

// ExecWithLastInsertId 执行INSERT并返回最后插入的ID
func (p *PgClient) ExecWithLastInsertId(ctx context.Context, query string, args ...interface{}) (*ExecResult, error) {
	// 检查是否是INSERT语句，并且有RETURNING子句
	upperQuery := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(upperQuery, "INSERT") && strings.Contains(upperQuery, "RETURNING") {
		var lastId int64
		err := p.db.QueryRowContext(ctx, query, args...).Scan(&lastId)
		if err != nil {
			return nil, fmt.Errorf("执行INSERT并获取ID失败: %w", err)
		}

		return &ExecResult{
			RowsAffected: 1,
			LastInsertId: lastId,
			Message:      "插入成功",
		}, nil
	}

	// 如果没有RETURNING子句，使用普通的Exec
	return p.Exec(ctx, query, args...)
}

// BeginTx 开始事务
func (p *PgClient) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return p.db.BeginTxx(ctx, nil)
}

// ListTables 列出数据库中的所有表
func (p *PgClient) ListTables(ctx context.Context, schema string) (*QueryResult, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT table_name, table_type, table_schema
		FROM information_schema.tables 
		WHERE table_schema = $1
		ORDER BY table_name
	`

	return p.Query(ctx, query, schema)
}

// ListColumns 列出表的所有列信息
func (p *PgClient) ListColumns(ctx context.Context, tableName, schema string) (*QueryResult, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			numeric_precision,
			numeric_scale
		FROM information_schema.columns 
		WHERE table_name = $1 AND table_schema = $2
		ORDER BY ordinal_position
	`

	return p.Query(ctx, query, tableName, schema)
}

// ListIndexes 列出表的所有索引
func (p *PgClient) ListIndexes(ctx context.Context, tableName, schema string) (*QueryResult, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT 
			i.indexname as index_name,
			i.indexdef as index_definition,
			'INDEX' as index_type
		FROM pg_indexes i
		WHERE i.tablename = $1 AND i.schemaname = $2
		ORDER BY i.indexname
	`

	return p.Query(ctx, query, tableName, schema)
}

// ListSchemas 列出数据库中的所有模式
func (p *PgClient) ListSchemas(ctx context.Context) (*QueryResult, error) {
	query := `
		SELECT schema_name 
		FROM information_schema.schemata 
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`

	return p.Query(ctx, query)
}
