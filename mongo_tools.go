package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"xz_mcp/db/mongodb_db"
)

// mongoOpTimeout 单次 MongoDB 操作的超时上限。
// MCP 走 stdio 单通道，一条慢查询挂住会导致所有工具都不响应，因此必须有硬上限。
const mongoOpTimeout = 30 * time.Second

// mongoOpCtx 为一次 MongoDB 操作派生带超时的 context。
// 例: ctx, cancel := mongoOpCtx(ctx); defer cancel() → 该操作最多执行 30s
func mongoOpCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, mongoOpTimeout)
}

// requireMongoClient 校验当前是否已有活动连接，未连接时返回统一提示。
func requireMongoClient() error {
	if mongoClient == nil {
		return fmt.Errorf("没有活动的MongoDB连接，请先执行 mongo_connect")
	}
	return nil
}

// registerMongoTools 注册 MongoDB 相关工具。
// 采用 4 工具通用式：连接 + 查询 + 聚合 + 原生命令。
// 插入/更新/删除/索引/库表管理等全部写操作与管理操作均通过 mongo_command 的原生命令协议完成。
func registerMongoTools(s *server.MCPServer) {
	// 1. mongo_connect - 动态连接到 MongoDB
	s.AddTool(
		mcp.NewTool("mongo_connect",
			mcp.WithDescription("连接到MongoDB服务器（动态传参）。支持两种形态：传 uri 一整条连接串，或分别传 host/port/username/password/auth_source。可选的 database 会成为后续操作的默认库。"),
			mcp.WithString("uri", mcp.Description("完整连接串，如 mongodb://user:pass@127.0.0.1:27017/?authSource=admin 或 mongodb+srv://... （传了则忽略下面的分离字段）")),
			mcp.WithString("host", mcp.Description("MongoDB服务器地址（默认: 127.0.0.1）")),
			mcp.WithNumber("port", mcp.DefaultNumber(27017), mcp.Description("MongoDB端口（默认: 27017）")),
			mcp.WithString("username", mcp.Description("用户名")),
			mcp.WithString("password", mcp.Description("密码")),
			mcp.WithString("auth_source", mcp.DefaultString("admin"), mcp.Description("认证数据库（默认: admin）")),
			mcp.WithString("database", mcp.Description("默认数据库名，设置后 find/aggregate/command 可不传 database")),
		),
		handleMongoConnect,
	)

	// 2. mongo_find - 查询文档
	s.AddTool(
		mcp.NewTool("mongo_find",
			mcp.WithDescription("查询MongoDB集合中的文档。filter/projection/sort 均为 Extended JSON 字符串——按 _id 查询须写成 {\"_id\":{\"$oid\":\"...\"}}，直接写字符串 id 匹配不到。"),
			mcp.WithString("collection", mcp.Required(), mcp.Description("集合名")),
			mcp.WithString("database", mcp.Description("数据库名（不传则使用 mongo_connect 时绑定的默认库）")),
			mcp.WithString("filter", mcp.Description("查询条件，Extended JSON 字符串，如 {\"age\":{\"$gt\":18}} 或 {\"_id\":{\"$oid\":\"68a1f2c9e1b2c3d4e5f60718\"}}（默认: {} 查全部）")),
			mcp.WithString("projection", mcp.Description("字段投影，Extended JSON 字符串，如 {\"_id\":1,\"name\":1}")),
			mcp.WithString("sort", mcp.Description("排序规则，Extended JSON 字符串，如 {\"created_at\":-1}（-1 倒序，1 正序）")),
			mcp.WithNumber("limit", mcp.DefaultNumber(100), mcp.Description("返回条数上限（默认: 100；显式传 0 表示不限制，大集合慎用）")),
			mcp.WithNumber("skip", mcp.DefaultNumber(0), mcp.Description("跳过条数，用于分页（默认: 0）")),
		),
		handleMongoFind,
	)

	// 3. mongo_aggregate - 聚合查询
	s.AddTool(
		mcp.NewTool("mongo_aggregate",
			mcp.WithDescription("执行MongoDB聚合管道。pipeline 必须是 Extended JSON 数组字符串，如 [{\"$match\":{...}},{\"$group\":{...}}]。若含 $out/$merge 阶段会写入集合，返回结果中会附带 warning。"),
			mcp.WithString("collection", mcp.Required(), mcp.Description("集合名")),
			mcp.WithString("database", mcp.Description("数据库名（不传则使用 mongo_connect 时绑定的默认库）")),
			mcp.WithString("pipeline", mcp.Required(), mcp.Description("聚合管道，Extended JSON 数组字符串，如 [{\"$group\":{\"_id\":null,\"total\":{\"$sum\":1}}}]")),
		),
		handleMongoAggregate,
	)

	// 4. mongo_command - 执行任意原生数据库命令（全权限入口）
	s.AddTool(
		mcp.NewTool("mongo_command",
			mcp.WithDescription("执行任意MongoDB原生数据库命令，是插入/更新/删除/建索引/库表管理/服务器管理的统一入口。"+
				"示例：插入 {\"insert\":\"users\",\"documents\":[{...}]}；更新 {\"update\":\"users\",\"updates\":[{\"q\":{...},\"u\":{\"$set\":{...}}}]}；"+
				"删数据 {\"delete\":\"users\",\"deletes\":[{\"q\":{},\"limit\":0}]}；建索引 {\"createIndexes\":\"users\",\"indexes\":[{\"key\":{\"email\":1},\"name\":\"idx_email\"}]}；"+
				"列库 {\"listDatabases\":1}（需 database:\"admin\"）。"+
				"注意：dropDatabase（删库）/ drop（删集合）/ shutdown（停实例）属于不可逆的结构级操作，必须同时传 confirm:true 才会执行；删数据等操作无需确认。"),
			mcp.WithString("command", mcp.Required(), mcp.Description("命令文档，Extended JSON 字符串，命令名必须是第一个字段")),
			mcp.WithString("database", mcp.Description("在哪个库执行（不传则使用默认库；listDatabases/serverStatus 等管理命令需传 admin）")),
			mcp.WithBoolean("confirm", mcp.Description("对 dropDatabase/drop/shutdown 这类不可逆结构级操作的人工确认，必须显式传 true 才会执行")),
		),
		handleMongoCommand,
	)
}

// handleMongoConnect MongoDB连接处理器。
// 支持 uri 与分离字段两种形态；重复连接时先断开旧连接，避免连接池泄漏。
func handleMongoConnect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	config := mongodb_db.MongoConfig{
		URI:        request.GetString("uri", ""),
		Host:       request.GetString("host", ""),
		Port:       request.GetInt("port", 27017),
		Username:   request.GetString("username", ""),
		Password:   request.GetString("password", ""),
		AuthSource: request.GetString("auth_source", "admin"),
		Database:   request.GetString("database", ""),
	}

	if strings.TrimSpace(config.URI) == "" && strings.TrimSpace(config.Host) == "" {
		return mcp.NewToolResultError("请提供 uri（完整连接串），或提供 host（分离字段形式）"), nil
	}

	// 已有连接先关闭，防止旧连接池常驻
	if mongoClient != nil {
		_ = mongoClient.Close(ctx)
		mongoClient = nil
	}

	connCtx, cancel := mongoOpCtx(ctx)
	defer cancel()

	client, err := mongodb_db.NewMongoClient(connCtx, config)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("连接失败: %v", err)), nil
	}
	mongoClient = client

	response := map[string]interface{}{
		"status":      "connected",
		"host":        config.Host,
		"port":        config.Port,
		"username":    config.Username,
		"auth_source": config.AuthSource,
		"database":    client.DefaultDB(),
	}
	if config.URI != "" {
		response["uri"] = config.URI
		// 用 uri 连接时分离字段为空，去掉以免造成误解
		delete(response, "host")
		delete(response, "port")
		delete(response, "username")
		delete(response, "auth_source")
	}

	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleMongoFind MongoDB查询处理器。
func handleMongoFind(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := requireMongoClient(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collection, err := request.RequireString("collection")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 三个 BSON 参数各自解析，失败时错误信息会带上是哪个参数写错了
	filter, err := mongodb_db.ParseExtJSONDoc("filter", request.GetString("filter", "{}"))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	projection, err := mongodb_db.ParseExtJSONDoc("projection", request.GetString("projection", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sort, err := mongodb_db.ParseExtJSONDoc("sort", request.GetString("sort", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 未传 limit 时取默认 100 条；显式传 0 才表示不限制
	limit := request.GetInt("limit", int(mongodb_db.DefaultFindLimit))
	skip := request.GetInt("skip", 0)

	opCtx, cancel := mongoOpCtx(ctx)
	defer cancel()

	result, err := mongoClient.Find(opCtx, request.GetString("database", ""), collection, mongodb_db.FindParams{
		Filter:     filter,
		Projection: projection,
		Sort:       sort,
		Limit:      int64(limit),
		Skip:       int64(skip),
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mongoQueryResultToText(result, "")
}

// handleMongoAggregate MongoDB聚合处理器。
func handleMongoAggregate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := requireMongoClient(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collection, err := request.RequireString("collection")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	pipelineStr, err := request.RequireString("pipeline")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pipeline, err := mongodb_db.ParseExtJSONArray("pipeline", pipelineStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opCtx, cancel := mongoOpCtx(ctx)
	defer cancel()

	result, warning, err := mongoClient.Aggregate(opCtx, request.GetString("database", ""), collection, pipeline)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mongoQueryResultToText(result, warning)
}

// handleMongoCommand MongoDB原生命令处理器。
// 结构级不可逆命令（dropDatabase/drop/shutdown）必须显式传 confirm:true 才执行。
func handleMongoCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := requireMongoClient(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	commandStr, err := request.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cmd, err := mongodb_db.ParseExtJSONDoc("command", commandStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(cmd) == 0 {
		return mcp.NewToolResultError("参数 command 不能为空文档，命令名必须是第一个字段，如 {\"listCollections\":1}"), nil
	}

	// 不可逆结构级操作拦截：未确认则直接拒绝，不执行任何操作
	if need, desc := mongodb_db.NeedsConfirm(cmd); need && !request.GetBool("confirm", false) {
		return mcp.NewToolResultError(fmt.Sprintf(
			"该操作为不可逆的结构级操作（%s：%s），需人工确认。确认无误后，在参数中加入 confirm: true 重新调用。",
			desc, mongodb_db.CommandName(cmd))), nil
	}

	opCtx, cancel := mongoOpCtx(ctx)
	defer cancel()

	result, err := mongoClient.RunCommand(opCtx, request.GetString("database", ""), cmd)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 游标类命令已取全量，用 results/count 呈现；普通命令原样返回响应文档
	var payload any = result.Raw
	if result.Cursor {
		payload = map[string]any{
			"results": result.Results,
			"count":   result.Count,
		}
	}

	text, err := mongodb_db.ToExtJSON(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(text), nil
}

// mongoQueryResultToText 把查询/聚合结果序列化为 Extended JSON 文本。
// 空结果稳定返回 {"data":[],"count":0}，不会出现 "data":null。
func mongoQueryResultToText(result *mongodb_db.QueryResult, warning string) (*mcp.CallToolResult, error) {
	payload := map[string]any{
		"data":  result.Data,
		"count": result.Count,
	}
	if warning != "" {
		payload["warning"] = warning
	}

	text, err := mongodb_db.ToExtJSON(payload)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(text), nil
}
