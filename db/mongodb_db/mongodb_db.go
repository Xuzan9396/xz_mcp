package mongodb_db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ============================================================================
// 配置与连接管理
// ============================================================================

// MongoConfig MongoDB 连接配置。
// 支持两种形态：URI 非空时直接使用；否则由 Host/Port/Username/Password/AuthSource 拼接。
type MongoConfig struct {
	URI        string `json:"uri,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"-"`
	AuthSource string `json:"auth_source,omitempty"`
	Database   string `json:"database,omitempty"`
}

// MongoClient MongoDB 客户端包装器，持有连接与默认数据库名。
type MongoClient struct {
	client    *mongo.Client
	config    MongoConfig
	defaultDB string
}

const (
	// ServerSelectionTimeout 服务器选择超时。驱动默认 30s，缩短到 5s 以便地址写错时快速失败。
	ServerSelectionTimeout = 5 * time.Second
	// ConnectTimeout 单次 TCP 建连超时。
	ConnectTimeout = 10 * time.Second
	// DefaultFindLimit Find 未指定 limit 时的默认上限，防止一次拉取整个集合撑爆内存。
	DefaultFindLimit int64 = 100
)

// BuildURI 根据配置构造 MongoDB 连接串。
// config.URI 非空时原样返回；否则用分离字段拼接，缺省值为 127.0.0.1:27017 与 authSource=admin。
// 例: {Host:"127.0.0.1", Port:27017, Username:"root", Password:"p@ss", AuthSource:"admin"}
//     → "mongodb://root:p%40ss@127.0.0.1:27017/?authSource=admin"（密码中的 @ 被转义为 %40）
func BuildURI(config MongoConfig) string {
	if strings.TrimSpace(config.URI) != "" {
		return strings.TrimSpace(config.URI)
	}

	host := config.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := config.Port
	if port == 0 {
		port = 27017
	}
	authSource := config.AuthSource
	if authSource == "" {
		authSource = "admin"
	}

	// 用户名密码可能含 @ : / 等 URI 保留字符，必须转义后再拼接
	credential := ""
	if config.Username != "" {
		credential = url.QueryEscape(config.Username)
		if config.Password != "" {
			credential += ":" + url.QueryEscape(config.Password)
		}
		credential += "@"
	}

	// 拼接示例: mongodb:// + "root:root123456@" + "127.0.0.1:27017" + "/?authSource=admin"
	return fmt.Sprintf("mongodb://%s%s:%d/?authSource=%s", credential, host, port, authSource)
}

// parseDBFromURI 从连接串路径段中提取数据库名。
// 例: "mongodb://h:27017/shop?authSource=admin" → "shop"；无路径段时返回 ""
func parseDBFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

// NewMongoClient 创建并验证 MongoDB 连接。
// v2 驱动的 Connect 是惰性的、不会立即建连，因此这里必须紧接一次 Ping 才能确认地址与凭据可用。
// 例: NewMongoClient(ctx, MongoConfig{Host:"127.0.0.1", Port:27017, Database:"shop"})
//     → 地址不通时约 5s 内返回错误，而不是等到首次查询才暴露
func NewMongoClient(ctx context.Context, config MongoConfig) (*MongoClient, error) {
	uri := BuildURI(config)

	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(ServerSelectionTimeout).
		SetConnectTimeout(ConnectTimeout)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("创建MongoDB客户端失败: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		// Ping 失败必须回收连接池，否则后台协程会持续重试
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("MongoDB连接测试失败: %w", err)
	}

	defaultDB := config.Database
	if defaultDB == "" {
		defaultDB = parseDBFromURI(uri)
	}

	return &MongoClient{
		client:    client,
		config:    config,
		defaultDB: defaultDB,
	}, nil
}

// Close 断开 MongoDB 连接并释放连接池。
func (m *MongoClient) Close(ctx context.Context) error {
	if m == nil || m.client == nil {
		return nil
	}
	return m.client.Disconnect(ctx)
}

// Ping 测试连接是否仍然可用。
func (m *MongoClient) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}

// DefaultDB 返回连接时绑定的默认数据库名，未绑定时为空串。
func (m *MongoClient) DefaultDB() string {
	return m.defaultDB
}

// Config 返回创建该客户端时使用的配置。
func (m *MongoClient) Config() MongoConfig {
	return m.config
}

// resolveDB 决定本次操作作用于哪个数据库：显式传入的优先，否则用连接时绑定的默认库。
// 例: defaultDB="shop" 时，resolveDB("") → shop 库；resolveDB("admin") → admin 库（用于 listDatabases 等管理命令）
func (m *MongoClient) resolveDB(dbName string) (*mongo.Database, error) {
	name := strings.TrimSpace(dbName)
	if name == "" {
		name = m.defaultDB
	}
	if name == "" {
		return nil, fmt.Errorf("未指定数据库：请在 mongo_connect 时提供 database，或在本次调用中传入 database 参数")
	}
	return m.client.Database(name), nil
}

// ============================================================================
// Extended JSON 转换层
//
// MCP 传入的是普通 JSON 文本，而 MongoDB 的 _id 默认是 ObjectID 类型、时间是日期类型。
// 普通 JSON 里的字符串 "68a1f2..." 不等于 ObjectID("68a1f2...")，按 _id 查会永远查不到。
// 因此所有 BSON 参数统一走 Extended JSON 解析，让 {"$oid":...} / {"$date":...} 能表达真实类型。
// ============================================================================

// truncate 截断过长文本用于错误提示，避免把整个大文档回显到错误信息里。
// 例: truncate("abcdefg", 3) → "abc...(已截断)"
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(已截断)"
}

// ParseExtJSONDoc 把 Extended JSON 文本解析为 bson.D 文档。
// paramName 仅用于错误提示，让调用方一眼看出是哪个参数写错了。
// 例: ParseExtJSONDoc("filter", `{"_id":{"$oid":"68a1f2c9e1b2c3d4e5f60718"}}`)
//     → bson.D{{"_id", ObjectID(68a1f2c9e1b2c3d4e5f60718)}}，该条件能真正命中文档
func ParseExtJSONDoc(paramName, s string) (bson.D, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "{}" {
		return bson.D{}, nil
	}

	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(trimmed), false, &doc); err != nil {
		return nil, fmt.Errorf("参数 %s 不是合法的 Extended JSON: %v；收到: %s",
			paramName, err, truncate(trimmed, 200))
	}
	return doc, nil
}

// ParseExtJSONArray 把 Extended JSON 文本解析为 bson.A 数组，用于聚合 pipeline。
// 先做首字符检查，是为了把「传成对象」这一高频错误变成一句人话，而不是驱动内部的编码错误。
// 例: ParseExtJSONArray("pipeline", `{"$match":{}}`)
//     → 报错「参数 pipeline 必须是 JSON 数组（形如 [{"$match":{...}}]）；收到: {"$match":{}}」
func ParseExtJSONArray(paramName, s string) (bson.A, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return bson.A{}, nil
	}
	if !strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("参数 %s 必须是 JSON 数组（形如 [{\"$match\":{...}}]）；收到: %s",
			paramName, truncate(trimmed, 200))
	}

	var arr bson.A
	if err := bson.UnmarshalExtJSON([]byte(trimmed), false, &arr); err != nil {
		return nil, fmt.Errorf("参数 %s 不是合法的 Extended JSON 数组: %v；收到: %s",
			paramName, err, truncate(trimmed, 200))
	}
	return arr, nil
}

// ToExtJSON 把 BSON 值序列化为 Extended JSON 文本（relaxed 模式，可读性优先）。
// 例: bson.M{"_id": ObjectID(...)} → `{"_id":{"$oid":"68a1f2c9e1b2c3d4e5f60718"}}`，
//     返回的 $oid 可以原样复制回 filter 里再次查询
func ToExtJSON(v any) (string, error) {
	data, err := bson.MarshalExtJSON(v, false, false)
	if err != nil {
		return "", fmt.Errorf("序列化为 Extended JSON 失败: %w", err)
	}
	return string(data), nil
}

// DocsToExtJSON 把文档切片序列化为 Extended JSON 数组文本。
// 空切片返回 "[]" 而不是 "null"，避免调用方把「没查到」误判成「出错了」。
func DocsToExtJSON(docs []bson.M) (string, error) {
	if len(docs) == 0 {
		return "[]", nil
	}
	return ToExtJSON(docs)
}

// ============================================================================
// 查询
// ============================================================================

// QueryResult 查询与聚合的统一返回结构。
type QueryResult struct {
	Data  []bson.M `json:"data"`
	Count int      `json:"count"`
}

// FindParams Find 的可选查询条件。Limit 为 0 表示不限制条数。
type FindParams struct {
	Filter     bson.D
	Projection bson.D
	Sort       bson.D
	Limit      int64
	Skip       int64
}

// Find 执行查询并一次性取回结果。
// 例: Find(ctx, "", "users", FindParams{Filter: bson.D{}, Sort: bson.D{{"age", -1}}, Limit: 20, Skip: 40})
//     → 按年龄倒序取第 3 页（跳过 40 条、取 20 条）
func (m *MongoClient) Find(ctx context.Context, dbName, collName string, p FindParams) (*QueryResult, error) {
	db, err := m.resolveDB(dbName)
	if err != nil {
		return nil, err
	}

	opts := options.Find()
	if p.Limit > 0 { // Limit == 0 表示调用方显式要求不限制
		opts = opts.SetLimit(p.Limit)
	}
	if p.Skip > 0 {
		opts = opts.SetSkip(p.Skip)
	}
	if len(p.Sort) > 0 {
		opts = opts.SetSort(p.Sort)
	}
	if len(p.Projection) > 0 {
		opts = opts.SetProjection(p.Projection)
	}

	filter := p.Filter
	if filter == nil {
		filter = bson.D{}
	}

	cursor, err := db.Collection(collName).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer cursor.Close(ctx) // 错误路径下也要归还服务端游标，否则要等 10 分钟才超时释放

	// 显式初始化为空切片：序列化后是 [] 而非 null
	results := []bson.M{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("读取查询结果失败: %w", err)
	}

	return &QueryResult{Data: results, Count: len(results)}, nil
}

// ============================================================================
// 聚合
// ============================================================================

// DetectWriteStages 检查 pipeline 是否含会写入数据的阶段。
// $out 会覆盖目标集合的全部内容、$merge 会写入或更新目标集合，两者都不是只读操作。
// 例: [{"$group":...},{"$out":"report"}] → 返回「警告：pipeline 含 $out 阶段，...」
func DetectWriteStages(pipeline bson.A) string {
	for _, stage := range pipeline {
		var keys []string
		switch s := stage.(type) {
		case bson.D:
			for _, e := range s {
				keys = append(keys, e.Key)
			}
		case bson.M:
			for k := range s {
				keys = append(keys, k)
			}
		case map[string]any:
			for k := range s {
				keys = append(keys, k)
			}
		}
		for _, k := range keys {
			if k == "$out" || k == "$merge" {
				return fmt.Sprintf("警告：pipeline 含 %s 阶段，聚合结果将写入集合并可能覆盖其原有数据", k)
			}
		}
	}
	return ""
}

// Aggregate 执行聚合管道。
// 第二个返回值是写入阶段警示（无则为空串），让调用方知道这次聚合并非只读。
// 例: Aggregate(ctx, "", "users", bson.A{bson.D{{"$group", bson.D{{"_id", nil}, {"total", bson.D{{"$sum", 1}}}}}}})
//     → QueryResult{Data:[{_id:null, total:2}], Count:1}, warning:""
func (m *MongoClient) Aggregate(ctx context.Context, dbName, collName string, pipeline bson.A) (*QueryResult, string, error) {
	db, err := m.resolveDB(dbName)
	if err != nil {
		return nil, "", err
	}

	warning := DetectWriteStages(pipeline)

	cursor, err := db.Collection(collName).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, warning, fmt.Errorf("聚合执行失败: %w", err)
	}
	defer cursor.Close(ctx)

	results := []bson.M{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, warning, fmt.Errorf("读取聚合结果失败: %w", err)
	}

	return &QueryResult{Data: results, Count: len(results)}, warning, nil
}

// ============================================================================
// 原生命令
// ============================================================================

// confirmRequiredCommands 需要人工确认才能执行的结构级不可逆命令。
// 键统一小写以便忽略大小写比对。
// 刻意不包含 delete / update / findAndModify / dropIndexes 等数据级操作——
// 按需求约定：删数据不需要确认，只有删集合、删库、停实例这类结构性破坏才要确认。
var confirmRequiredCommands = map[string]string{
	"dropdatabase": "删除整个数据库",
	"drop":         "删除集合",
	"shutdown":     "关闭 MongoDB 实例",
}

// CommandName 取出命令名。MongoDB 的命令名固定是命令文档的第一个字段。
// 例: bson.D{{"drop","users"}} → "drop"；bson.D{{"insert","users"},{"documents",...}} → "insert"
func CommandName(cmd bson.D) string {
	if len(cmd) == 0 {
		return ""
	}
	return cmd[0].Key
}

// NeedsConfirm 判断命令是否属于需要人工确认的结构级不可逆操作。
// 例: NeedsConfirm(bson.D{{"drop","users"}}) → (true, "删除集合")
//     NeedsConfirm(bson.D{{"delete","users"},...}) → (false, "")  // 删数据无需确认
func NeedsConfirm(cmd bson.D) (bool, string) {
	desc, ok := confirmRequiredCommands[strings.ToLower(CommandName(cmd))]
	return ok, desc
}

// CommandResult 原生命令的执行结果。
// Cursor 为 true 时结果在 Results 中（已取全量）；为 false 时结果在 Raw 中。
type CommandResult struct {
	Raw     bson.M   `json:"raw,omitempty"`
	Results []bson.M `json:"results,omitempty"`
	Count   int      `json:"count,omitempty"`
	Cursor  bool     `json:"-"`
}

// RunCommand 执行任意 MongoDB 原生数据库命令。
// listCollections / listIndexes / find 这类命令的响应是游标形式，
// 普通 RunCommand 只能拿到 firstBatch（约 101 条），剩余部分会静默丢失；
// 因此这里先探测响应里有没有 cursor 字段，有就改用 RunCommandCursor 取全量。
// 例: RunCommand(ctx, "", bson.D{{"listCollections", 1}}) 在有 300 个集合的库上
//     → Results 含全部 300 项、Count=300，而不是只有前 101 项
func (m *MongoClient) RunCommand(ctx context.Context, dbName string, cmd bson.D) (*CommandResult, error) {
	db, err := m.resolveDB(dbName)
	if err != nil {
		return nil, err
	}

	raw, err := db.RunCommand(ctx, cmd).Raw()
	if err != nil {
		return nil, fmt.Errorf("命令执行失败: %w", err)
	}

	// LookupErr 返回 nil error 即代表响应顶层存在 cursor 字段
	if _, lookupErr := raw.LookupErr("cursor"); lookupErr != nil {
		// 非游标类命令：直接返回完整响应
		var doc bson.M
		if err := bson.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("解析命令响应失败: %w", err)
		}
		return &CommandResult{Raw: doc, Cursor: false}, nil
	}

	// 游标类命令：重新执行以拿到可迭代游标，取回全部批次
	cursor, err := db.RunCommandCursor(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("命令游标获取失败: %w", err)
	}
	defer cursor.Close(ctx)

	results := []bson.M{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("读取命令结果失败: %w", err)
	}

	return &CommandResult{Results: results, Count: len(results), Cursor: true}, nil
}
