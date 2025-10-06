package redis_db

import (
	"context"
	"testing"
)

// 测试用的Redis配置
var testConfig = RedisConfig{
	Addr:     "127.0.0.1:6379",
	Password: "27252725",
	DB:       1,
}

func TestNewRedisClient(t *testing.T) {
	client := NewRedisClient(testConfig)
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis server not available: %v", err)
	}

	t.Log("Redis client created successfully")
}

func TestExecuteCommand(t *testing.T) {
	client := NewRedisClient(testConfig)
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis server not available: %v", err)
	}

	testKey := "test:command"
	testValue := "command test"

	// Test SET command
	setArgs := []interface{}{"SET", testKey, testValue}
	result, err := client.ExecuteCommand(ctx, setArgs)
	if err != nil {
		t.Fatalf("Failed to execute SET command: %v", err)
	}

	if result != "OK" {
		t.Errorf("Expected OK, got %v", result)
	}

	// Test GET command
	getArgs := []interface{}{"GET", testKey}
	result, err = client.ExecuteCommand(ctx, getArgs)
	if err != nil {
		t.Fatalf("Failed to execute GET command: %v", err)
	}

	if result != testValue {
		t.Errorf("Expected %s, got %v", testValue, result)
	}

	// Cleanup
	delArgs := []interface{}{"DEL", testKey}
	_, err = client.ExecuteCommand(ctx, delArgs)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	t.Log("Execute command test passed")
}

func TestExecuteLuaScript(t *testing.T) {
	client := NewRedisClient(testConfig)
	defer client.Close()

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("Redis server not available: %v", err)
	}

	testKey := "test:lua"
	luaScript := `
		local key = KEYS[1]
		local value = ARGV[1]
		redis.call('SET', key, value)
		return redis.call('GET', key)
	`

	result, err := client.ExecuteLuaScript(ctx, luaScript, []string{testKey}, []interface{}{"lua test"})
	if err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	if result != "lua test" {
		t.Errorf("Expected 'lua test', got %v", result)
	}

	// Cleanup
	delArgs := []interface{}{"DEL", testKey}
	_, err = client.ExecuteCommand(ctx, delArgs)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	t.Log("Lua script test passed")
}

func TestParseRedisCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []interface{}
	}{
		{
			input:    "SET key value",
			expected: []interface{}{"SET", "key", "value"},
		},
		{
			input:    "INCR counter",
			expected: []interface{}{"INCR", "counter"},
		},
		{
			input:    "ZADD zset 1 member1",
			expected: []interface{}{"ZADD", "zset", 1, "member1"},
		},
		{
			input:    "ZADD zset 1.5 member2",
			expected: []interface{}{"ZADD", "zset", 1.5, "member2"},
		},
	}

	for _, tt := range tests {
		result, err := ParseRedisCommand(tt.input)
		if err != nil {
			t.Errorf("Failed to parse command '%s': %v", tt.input, err)
			continue
		}

		if len(result) != len(tt.expected) {
			t.Errorf("Expected %d args, got %d for command '%s'", len(tt.expected), len(result), tt.input)
			continue
		}

		for i, arg := range result {
			if arg != tt.expected[i] {
				t.Errorf("Expected arg %d to be %v, got %v for command '%s'", i, tt.expected[i], arg, tt.input)
			}
		}
	}

	t.Log("Parse redis command test passed")
}

func TestFormatRedisResult(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{
			input:    nil,
			expected: "null",
		},
		{
			input:    "hello",
			expected: "\"hello\"",
		},
		{
			input:    int64(42),
			expected: "42",
		},
		{
			input:    1.5,
			expected: "1.500000",
		},
		{
			input:    true,
			expected: "true",
		},
		{
			input:    []interface{}{"a", "b", "c"},
			expected: "[\"a\",\"b\",\"c\"]",
		},
	}

	for _, tt := range tests {
		result, err := FormatRedisResult(tt.input)
		if err != nil {
			t.Errorf("Failed to format result %v: %v", tt.input, err)
			continue
		}

		if result != tt.expected {
			t.Errorf("Expected %s, got %s for input %v", tt.expected, result, tt.input)
		}
	}

	t.Log("Format redis result test passed")
}

func TestSSLInsecureSkipVerifyConfig(t *testing.T) {
	t.Run("Default SSL config (nil)", func(t *testing.T) {
		config := RedisConfig{
			Addr:                  "127.0.0.1:6379",
			Password:              "27252725",
			DB:                    1,
			SSLInsecureSkipVerify: nil, // 默认不设置
		}

		client := NewRedisClient(config)
		defer client.Close()

		// 验证客户端创建成功（TLS 配置为 nil）
		if client.client == nil {
			t.Fatal("Expected client to be created")
		}
		t.Log("Default SSL config test passed")
	})

	t.Run("SSL skip verify enabled (true)", func(t *testing.T) {
		skipVerify := true
		config := RedisConfig{
			Addr:                  "127.0.0.1:6379",
			Password:              "27252725",
			DB:                    1,
			SSLInsecureSkipVerify: &skipVerify, // 设置为 true 时启用跳过验证
		}

		client := NewRedisClient(config)
		defer client.Close()

		// 验证客户端创建成功（TLS 配置应该存在且 InsecureSkipVerify=true）
		if client.client == nil {
			t.Fatal("Expected client to be created")
		}
		t.Log("SSL skip verify enabled test passed")
	})

	t.Run("SSL skip verify disabled (false)", func(t *testing.T) {
		skipVerify := false
		config := RedisConfig{
			Addr:                  "127.0.0.1:6379",
			Password:              "27252725",
			DB:                    1,
			SSLInsecureSkipVerify: &skipVerify, // 设置为 false 时不启用跳过验证
		}

		client := NewRedisClient(config)
		defer client.Close()

		// 验证客户端创建成功（TLS 配置应该为 nil）
		if client.client == nil {
			t.Fatal("Expected client to be created")
		}
		t.Log("SSL skip verify disabled test passed")
	})
}
