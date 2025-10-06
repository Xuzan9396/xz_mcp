package redis_db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

// RedisConfig Redis配置结构
type RedisConfig struct {
	Addr                  string `json:"addr"`
	Password              string `json:"password"`
	DB                    int    `json:"db"`
	SSLInsecureSkipVerify *bool  `json:"ssl_insecure_skip_verify,omitempty"`
}

// RedisClient Redis客户端包装器
type RedisClient struct {
	client *redis.Client
	config RedisConfig
}

// NewRedisClient 创建新的Redis客户端
func NewRedisClient(config RedisConfig) *RedisClient {
	options := &redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}

	// 配置 TLS，如果指定了 ssl_insecure_skip_verify 为 true 时跳过SSL验证
	if config.SSLInsecureSkipVerify != nil && *config.SSLInsecureSkipVerify == true {
		options.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rdb := redis.NewClient(options)

	return &RedisClient{
		client: rdb,
		config: config,
	}
}

// Close 关闭Redis连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Ping 测试Redis连接
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// ExecuteCommand 执行Redis命令
func (r *RedisClient) ExecuteCommand(ctx context.Context, cmdArgs []interface{}) (interface{}, error) {
	if len(cmdArgs) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := r.client.Do(ctx, cmdArgs...)
	result, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("redis command failed: %w", err)
	}

	return result, nil
}

// ExecuteLuaScript 执行Lua脚本
func (r *RedisClient) ExecuteLuaScript(ctx context.Context, script string, keys []string, args []interface{}) (interface{}, error) {
	cmd := r.client.Eval(ctx, script, keys, args...)
	result, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("lua script execution failed: %w", err)
	}

	return result, nil
}

// ParseRedisCommand 解析Redis命令字符串
func ParseRedisCommand(cmdStr string) ([]interface{}, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil, fmt.Errorf("empty command")
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	args := make([]interface{}, len(parts))
	for i, part := range parts {
		// 尝试解析为数字
		if intVal, err := strconv.Atoi(part); err == nil {
			args[i] = intVal
		} else if floatVal, err := strconv.ParseFloat(part, 64); err == nil {
			args[i] = floatVal
		} else {
			args[i] = part
		}
	}

	return args, nil
}

// FormatRedisResult 格式化Redis结果为JSON
func FormatRedisResult(result interface{}) (string, error) {
	switch v := result.(type) {
	case nil:
		return "null", nil
	case string:
		return fmt.Sprintf("\"%s\"", v), nil
	case []byte:
		return fmt.Sprintf("\"%s\"", string(v)), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case []interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal slice: %w", err)
		}
		return string(jsonBytes), nil
	case map[string]interface{}:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal map: %w", err)
		}
		return string(jsonBytes), nil
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(jsonBytes), nil
	}
}
