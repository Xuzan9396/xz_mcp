package mongodb_db

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// 测试配置。连接信息见 .xz_planning/phases/1.新增MongoDB动态连接与全权限操作工具/tests/.env.local
// 全部连库测试只在 xz_mcp_test 库内进行，跑完自行清理，不触碰其他库。
var testConfig = MongoConfig{
	Host:       "127.0.0.1",
	Port:       27017,
	Username:   "root",
	Password:   "root123456",
	AuthSource: "admin",
	Database:   "xz_mcp_test",
}

const testCollection = "test_users"

// ============================================================================
// 纯逻辑测试（不需要 MongoDB 实例）
// ============================================================================

func TestBuildURI(t *testing.T) {
	tests := []struct {
		name   string
		config MongoConfig
		expect string
	}{
		{
			name:   "URI 优先于分离字段",
			config: MongoConfig{URI: "mongodb+srv://u:p@cluster0.example.mongodb.net/", Host: "1.2.3.4", Port: 12345},
			expect: "mongodb+srv://u:p@cluster0.example.mongodb.net/",
		},
		{
			name:   "分离字段完整拼接",
			config: MongoConfig{Host: "127.0.0.1", Port: 27017, Username: "root", Password: "root123456", AuthSource: "admin"},
			expect: "mongodb://root:root123456@127.0.0.1:27017/?authSource=admin",
		},
		{
			name:   "默认值填充: host/port/authSource 全部缺省",
			config: MongoConfig{Username: "u", Password: "p"},
			expect: "mongodb://u:p@127.0.0.1:27017/?authSource=admin",
		},
		{
			name:   "无凭据时不拼 @ 段",
			config: MongoConfig{Host: "10.0.0.1", Port: 27018},
			expect: "mongodb://10.0.0.1:27018/?authSource=admin",
		},
		{
			name:   "密码含 URI 保留字符需转义: p@ss:w/d → p%40ss%3Aw%2Fd",
			config: MongoConfig{Host: "127.0.0.1", Port: 27017, Username: "root", Password: "p@ss:w/d"},
			expect: "mongodb://root:p%40ss%3Aw%2Fd@127.0.0.1:27017/?authSource=admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildURI(tt.config); got != tt.expect {
				t.Errorf("BuildURI() = %s, 期望 %s", got, tt.expect)
			}
		})
	}
}

func TestParseDBFromURI(t *testing.T) {
	tests := []struct {
		uri    string
		expect string
	}{
		{"mongodb://h:27017/shop?authSource=admin", "shop"},
		{"mongodb://h:27017/?authSource=admin", ""},
		{"mongodb://h:27017", ""},
	}
	for _, tt := range tests {
		if got := parseDBFromURI(tt.uri); got != tt.expect {
			t.Errorf("parseDBFromURI(%s) = %q, 期望 %q", tt.uri, got, tt.expect)
		}
	}
}

func TestParseExtJSONDoc(t *testing.T) {
	t.Run("空串与空文档返回空 bson.D", func(t *testing.T) {
		for _, s := range []string{"", "  ", "{}"} {
			doc, err := ParseExtJSONDoc("filter", s)
			if err != nil {
				t.Fatalf("解析 %q 失败: %v", s, err)
			}
			if len(doc) != 0 {
				t.Errorf("期望空文档，实际 %v", doc)
			}
		}
	})

	t.Run("$oid 解析为 ObjectID 而非字符串", func(t *testing.T) {
		doc, err := ParseExtJSONDoc("filter", `{"_id":{"$oid":"68a1f2c9e1b2c3d4e5f60718"}}`)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(doc) != 1 || doc[0].Key != "_id" {
			t.Fatalf("期望单字段 _id，实际 %v", doc)
		}
		oid, ok := doc[0].Value.(bson.ObjectID)
		if !ok {
			t.Fatalf("期望 _id 类型为 bson.ObjectID，实际 %T —— 这会导致按 _id 查询永远查不到", doc[0].Value)
		}
		if oid.Hex() != "68a1f2c9e1b2c3d4e5f60718" {
			t.Errorf("ObjectID 内容不符: %s", oid.Hex())
		}
	})

	t.Run("$date 解析为时间类型", func(t *testing.T) {
		doc, err := ParseExtJSONDoc("filter", `{"created_at":{"$date":"2026-01-01T00:00:00Z"}}`)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if _, ok := doc[0].Value.(bson.DateTime); !ok {
			t.Errorf("期望 bson.DateTime，实际 %T", doc[0].Value)
		}
	})

	t.Run("非法 JSON 的错误信息带参数名与原文", func(t *testing.T) {
		_, err := ParseExtJSONDoc("filter", `{"_id": }`)
		if err == nil {
			t.Fatal("期望报错，实际成功")
		}
		if !strings.Contains(err.Error(), "filter") {
			t.Errorf("错误信息应含参数名 filter，实际: %v", err)
		}
		if !strings.Contains(err.Error(), `{"_id": }`) {
			t.Errorf("错误信息应含原始片段，实际: %v", err)
		}
	})
}

func TestParseExtJSONArray(t *testing.T) {
	t.Run("合法数组解析出正确长度", func(t *testing.T) {
		arr, err := ParseExtJSONArray("pipeline", `[{"$match":{"age":25}},{"$group":{"_id":null}}]`)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(arr) != 2 {
			t.Errorf("期望 2 个阶段，实际 %d", len(arr))
		}
	})

	t.Run("传对象而非数组时给出可读错误", func(t *testing.T) {
		_, err := ParseExtJSONArray("pipeline", `{"$match":{"age":25}}`)
		if err == nil {
			t.Fatal("期望报错，实际成功")
		}
		if !strings.Contains(err.Error(), "必须是 JSON 数组") {
			t.Errorf("错误信息应说明必须是数组，实际: %v", err)
		}
		if !strings.Contains(err.Error(), "pipeline") {
			t.Errorf("错误信息应含参数名 pipeline，实际: %v", err)
		}
	})

	t.Run("空串返回空数组", func(t *testing.T) {
		arr, err := ParseExtJSONArray("pipeline", "")
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(arr) != 0 {
			t.Errorf("期望空数组，实际 %v", arr)
		}
	})
}

func TestNeedsConfirm(t *testing.T) {
	// 需确认：结构级不可逆操作
	needConfirm := []string{"dropDatabase", "drop", "shutdown", "DROP", "DropDatabase"}
	for _, name := range needConfirm {
		need, desc := NeedsConfirm(bson.D{{Key: name, Value: "test_users"}})
		if !need {
			t.Errorf("命令 %s 应该需要确认", name)
		}
		if desc == "" {
			t.Errorf("命令 %s 应该有中文描述", name)
		}
	}

	// 无需确认：数据级操作（按需求约定「删数据不用确认」）
	noConfirm := []string{"delete", "deleteMany", "update", "insert", "find",
		"findAndModify", "dropIndexes", "createIndexes", "listCollections", "listDatabases"}
	for _, name := range noConfirm {
		need, _ := NeedsConfirm(bson.D{{Key: name, Value: "test_users"}})
		if need {
			t.Errorf("命令 %s 属于数据级操作，不应要求确认", name)
		}
	}

	t.Run("空命令不报需确认", func(t *testing.T) {
		if need, _ := NeedsConfirm(bson.D{}); need {
			t.Error("空命令不应要求确认")
		}
	})
}

func TestCommandName(t *testing.T) {
	if got := CommandName(bson.D{{Key: "insert", Value: "users"}, {Key: "documents", Value: bson.A{}}}); got != "insert" {
		t.Errorf("CommandName = %q, 期望 insert", got)
	}
	if got := CommandName(bson.D{}); got != "" {
		t.Errorf("空文档的 CommandName 应为空串，实际 %q", got)
	}
}

func TestDetectWriteStages(t *testing.T) {
	t.Run("含 $out 返回警示", func(t *testing.T) {
		pipeline := bson.A{
			bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}}}},
			bson.D{{Key: "$out", Value: "report"}},
		}
		if w := DetectWriteStages(pipeline); w == "" || !strings.Contains(w, "$out") {
			t.Errorf("期望 $out 警示，实际 %q", w)
		}
	})

	t.Run("含 $merge 返回警示", func(t *testing.T) {
		pipeline := bson.A{bson.M{"$merge": "target"}}
		if w := DetectWriteStages(pipeline); w == "" || !strings.Contains(w, "$merge") {
			t.Errorf("期望 $merge 警示，实际 %q", w)
		}
	})

	t.Run("只读 pipeline 无警示", func(t *testing.T) {
		pipeline := bson.A{
			bson.D{{Key: "$match", Value: bson.D{{Key: "age", Value: 25}}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "age", Value: -1}}}},
		}
		if w := DetectWriteStages(pipeline); w != "" {
			t.Errorf("只读 pipeline 不应有警示，实际 %q", w)
		}
	})
}

func TestDocsToExtJSON(t *testing.T) {
	t.Run("空切片返回 [] 而非 null", func(t *testing.T) {
		got, err := DocsToExtJSON([]bson.M{})
		if err != nil {
			t.Fatalf("序列化失败: %v", err)
		}
		if got != "[]" {
			t.Errorf("期望 []，实际 %s", got)
		}
	})

	t.Run("nil 切片同样返回 []", func(t *testing.T) {
		got, err := DocsToExtJSON(nil)
		if err != nil {
			t.Fatalf("序列化失败: %v", err)
		}
		if got != "[]" {
			t.Errorf("期望 []，实际 %s", got)
		}
	})
}

// ============================================================================
// 连库测试（需要本地 MongoDB 实例）
// ============================================================================

// newTestClient 建立测试连接，连不上则跳过该测试。
func newTestClient(t *testing.T) (*MongoClient, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	client, err := NewMongoClient(ctx, testConfig)
	if err != nil {
		t.Skipf("MongoDB 不可用，跳过: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close(context.Background())
	})
	return client, ctx
}

func TestMongoClient_CRUD(t *testing.T) {
	client, ctx := newTestClient(t)

	// 起点先清干净，避免上一次跑残留数据干扰断言
	_, _ = client.RunCommand(ctx, "", bson.D{{Key: "drop", Value: testCollection}})

	// 1. 插入 3 条
	insertRes, err := client.RunCommand(ctx, "", bson.D{
		{Key: "insert", Value: testCollection},
		{Key: "documents", Value: bson.A{
			bson.D{{Key: "name", Value: "张三"}, {Key: "age", Value: 25}},
			bson.D{{Key: "name", Value: "李四"}, {Key: "age", Value: 31}},
			bson.D{{Key: "name", Value: "王五"}, {Key: "age", Value: 28}},
		}},
	})
	if err != nil {
		t.Fatalf("插入失败: %v", err)
	}
	if n, _ := insertRes.Raw["n"].(int32); n != 3 {
		t.Fatalf("期望插入 3 条，实际 %v", insertRes.Raw["n"])
	}

	// 2. 查全部，校验条数
	all, err := client.Find(ctx, "", testCollection, FindParams{Limit: DefaultFindLimit})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if all.Count != 3 {
		t.Fatalf("期望查到 3 条，实际 %d", all.Count)
	}

	// 3. 按 ObjectId 精确查询 —— Extended JSON 的核心价值
	oid, ok := all.Data[0]["_id"].(bson.ObjectID)
	if !ok {
		t.Fatalf("_id 类型异常: %T", all.Data[0]["_id"])
	}
	byID, err := client.Find(ctx, "", testCollection, FindParams{
		Filter: bson.D{{Key: "_id", Value: oid}},
	})
	if err != nil {
		t.Fatalf("按 _id 查询失败: %v", err)
	}
	if byID.Count != 1 {
		t.Fatalf("按 _id 查询期望命中 1 条，实际 %d", byID.Count)
	}

	// 3b. 走 ParseExtJSONDoc 的完整链路：{"$oid":"..."} 同样能命中
	extFilter, err := ParseExtJSONDoc("filter", `{"_id":{"$oid":"`+oid.Hex()+`"}}`)
	if err != nil {
		t.Fatalf("ExtJSON 解析失败: %v", err)
	}
	byExtID, err := client.Find(ctx, "", testCollection, FindParams{Filter: extFilter})
	if err != nil {
		t.Fatalf("按 ExtJSON _id 查询失败: %v", err)
	}
	if byExtID.Count != 1 {
		t.Fatalf("按 $oid 查询期望命中 1 条，实际 %d —— Extended JSON 链路有问题", byExtID.Count)
	}

	// 4. 排序 + 分页：按年龄倒序应为 31, 28, 25；跳过 1 取 1 → 28
	paged, err := client.Find(ctx, "", testCollection, FindParams{
		Sort:  bson.D{{Key: "age", Value: -1}},
		Limit: 1,
		Skip:  1,
	})
	if err != nil {
		t.Fatalf("排序分页查询失败: %v", err)
	}
	if paged.Count != 1 {
		t.Fatalf("期望 1 条，实际 %d", paged.Count)
	}
	if age, _ := paged.Data[0]["age"].(int32); age != 28 {
		t.Errorf("倒序跳过 1 条后期望 age=28，实际 %v", paged.Data[0]["age"])
	}

	// 5. 投影：只返回 name
	projected, err := client.Find(ctx, "", testCollection, FindParams{
		Projection: bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 0}},
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("投影查询失败: %v", err)
	}
	if _, exists := projected.Data[0]["age"]; exists {
		t.Errorf("投影应排除 age 字段，实际返回 %v", projected.Data[0])
	}

	// 6. 聚合统计
	agg, warning, err := client.Aggregate(ctx, "", testCollection, bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "avgAge", Value: bson.D{{Key: "$avg", Value: "$age"}}},
		}}},
	})
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	if warning != "" {
		t.Errorf("只读聚合不应有警示，实际: %s", warning)
	}
	if total, _ := agg.Data[0]["total"].(int32); total != 3 {
		t.Errorf("聚合期望 total=3，实际 %v", agg.Data[0]["total"])
	}

	// 7. 建索引
	if _, err := client.RunCommand(ctx, "", bson.D{
		{Key: "createIndexes", Value: testCollection},
		{Key: "indexes", Value: bson.A{
			bson.D{{Key: "key", Value: bson.D{{Key: "age", Value: 1}}}, {Key: "name", Value: "idx_age"}},
		}},
	}); err != nil {
		t.Fatalf("建索引失败: %v", err)
	}

	// 8. listIndexes 是游标类命令，应走 RunCommandCursor 取全量
	idxRes, err := client.RunCommand(ctx, "", bson.D{{Key: "listIndexes", Value: testCollection}})
	if err != nil {
		t.Fatalf("列索引失败: %v", err)
	}
	if !idxRes.Cursor {
		t.Error("listIndexes 应被识别为游标类命令")
	}
	if idxRes.Count < 2 {
		t.Errorf("期望至少 2 个索引（_id + idx_age），实际 %d", idxRes.Count)
	}

	// 9. 删数据（数据级操作，业务上无需确认）
	delRes, err := client.RunCommand(ctx, "", bson.D{
		{Key: "delete", Value: testCollection},
		{Key: "deletes", Value: bson.A{bson.D{{Key: "q", Value: bson.D{}}, {Key: "limit", Value: 0}}}},
	})
	if err != nil {
		t.Fatalf("删除数据失败: %v", err)
	}
	if n, _ := delRes.Raw["n"].(int32); n != 3 {
		t.Errorf("期望删除 3 条，实际 %v", delRes.Raw["n"])
	}

	// 10. 删集合，清理现场
	if _, err := client.RunCommand(ctx, "", bson.D{{Key: "drop", Value: testCollection}}); err != nil {
		t.Fatalf("删除集合失败: %v", err)
	}
}

func TestMongoClient_EmptyResult(t *testing.T) {
	client, ctx := newTestClient(t)

	result, err := client.Find(ctx, "", "collection_that_does_not_exist_12345", FindParams{Limit: 10})
	if err != nil {
		t.Fatalf("查询不存在的集合不应报错: %v", err)
	}
	if result.Data == nil {
		t.Fatal("Data 不应为 nil —— 会被序列化成 null，调用方容易误判为出错")
	}
	if result.Count != 0 {
		t.Errorf("期望 count=0，实际 %d", result.Count)
	}

	// 验证序列化结果确实是 [] 而不是 null
	text, err := DocsToExtJSON(result.Data)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if text != "[]" {
		t.Errorf("空结果应序列化为 []，实际 %s", text)
	}
}

func TestMongoClient_ResolveDB(t *testing.T) {
	client, ctx := newTestClient(t)

	// 显式指定 admin 库执行管理命令（默认库是 xz_mcp_test）
	res, err := client.RunCommand(ctx, "admin", bson.D{{Key: "listDatabases", Value: 1}})
	if err != nil {
		t.Fatalf("在 admin 库执行 listDatabases 失败: %v", err)
	}
	if res.Raw["databases"] == nil {
		t.Error("listDatabases 应返回 databases 字段")
	}

	// 默认库应为配置中的 xz_mcp_test
	if client.DefaultDB() != "xz_mcp_test" {
		t.Errorf("默认库期望 xz_mcp_test，实际 %s", client.DefaultDB())
	}

	// 无默认库且未传 database 时应报错
	noDefault := &MongoClient{client: client.client, defaultDB: ""}
	if _, err := noDefault.resolveDB(""); err == nil {
		t.Error("无默认库且未传 database 时应报错")
	}
}
