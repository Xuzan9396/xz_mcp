package redis_db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

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

// GetInfo 获取Redis信息
func (r *RedisClient) GetInfo(ctx context.Context, section string) (string, error) {
	cmd := r.client.Info(ctx, section)
	return cmd.Result()
}

// GetDBSize 获取数据库大小
func (r *RedisClient) GetDBSize(ctx context.Context) (int64, error) {
	cmd := r.client.DBSize(ctx)
	return cmd.Result()
}

// GetKeys 获取匹配的keys
func (r *RedisClient) GetKeys(ctx context.Context, pattern string) ([]string, error) {
	cmd := r.client.Keys(ctx, pattern)
	return cmd.Result()
}

// GetType 获取key的类型
func (r *RedisClient) GetType(ctx context.Context, key string) (string, error) {
	cmd := r.client.Type(ctx, key)
	return cmd.Result()
}

// GetTTL 获取key的过期时间
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	cmd := r.client.TTL(ctx, key)
	return cmd.Result()
}

// SetExpire 设置key的过期时间
func (r *RedisClient) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	cmd := r.client.Expire(ctx, key, expiration)
	return cmd.Err()
}

// Delete 删除key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) (int64, error) {
	cmd := r.client.Del(ctx, keys...)
	return cmd.Result()
}

// Exists 检查key是否存在
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	cmd := r.client.Exists(ctx, keys...)
	return cmd.Result()
}

// String operations
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	cmd := r.client.Set(ctx, key, value, expiration)
	return cmd.Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	cmd := r.client.Get(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	cmd := r.client.MGet(ctx, keys...)
	return cmd.Result()
}

func (r *RedisClient) MSet(ctx context.Context, pairs ...interface{}) error {
	cmd := r.client.MSet(ctx, pairs...)
	return cmd.Err()
}

func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	cmd := r.client.Incr(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	cmd := r.client.IncrBy(ctx, key, value)
	return cmd.Result()
}

func (r *RedisClient) Decr(ctx context.Context, key string) (int64, error) {
	cmd := r.client.Decr(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	cmd := r.client.DecrBy(ctx, key, value)
	return cmd.Result()
}

// Hash operations
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := r.client.HSet(ctx, key, values...)
	return cmd.Result()
}

func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	cmd := r.client.HGet(ctx, key, field)
	return cmd.Result()
}

func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	cmd := r.client.HGetAll(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	cmd := r.client.HDel(ctx, key, fields...)
	return cmd.Result()
}

func (r *RedisClient) HExists(ctx context.Context, key, field string) (bool, error) {
	cmd := r.client.HExists(ctx, key, field)
	return cmd.Result()
}

func (r *RedisClient) HKeys(ctx context.Context, key string) ([]string, error) {
	cmd := r.client.HKeys(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) HLen(ctx context.Context, key string) (int64, error) {
	cmd := r.client.HLen(ctx, key)
	return cmd.Result()
}

// List operations
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := r.client.LPush(ctx, key, values...)
	return cmd.Result()
}

func (r *RedisClient) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := r.client.RPush(ctx, key, values...)
	return cmd.Result()
}

func (r *RedisClient) LPop(ctx context.Context, key string) (string, error) {
	cmd := r.client.LPop(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) RPop(ctx context.Context, key string) (string, error) {
	cmd := r.client.RPop(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	cmd := r.client.LRange(ctx, key, start, stop)
	return cmd.Result()
}

func (r *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	cmd := r.client.LLen(ctx, key)
	return cmd.Result()
}

// Set operations
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	cmd := r.client.SAdd(ctx, key, members...)
	return cmd.Result()
}

func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	cmd := r.client.SMembers(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	cmd := r.client.SRem(ctx, key, members...)
	return cmd.Result()
}

func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	cmd := r.client.SIsMember(ctx, key, member)
	return cmd.Result()
}

func (r *RedisClient) SCard(ctx context.Context, key string) (int64, error) {
	cmd := r.client.SCard(ctx, key)
	return cmd.Result()
}

// Sorted Set operations
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	cmd := r.client.ZAdd(ctx, key, members...)
	return cmd.Result()
}

func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	cmd := r.client.ZRange(ctx, key, start, stop)
	return cmd.Result()
}

func (r *RedisClient) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	cmd := r.client.ZRangeWithScores(ctx, key, start, stop)
	return cmd.Result()
}

func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	cmd := r.client.ZRem(ctx, key, members...)
	return cmd.Result()
}

func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	cmd := r.client.ZCard(ctx, key)
	return cmd.Result()
}

func (r *RedisClient) ZScore(ctx context.Context, key, member string) (float64, error) {
	cmd := r.client.ZScore(ctx, key, member)
	return cmd.Result()
}

// Utility functions
func (r *RedisClient) FlushDB(ctx context.Context) error {
	cmd := r.client.FlushDB(ctx)
	return cmd.Err()
}

func (r *RedisClient) FlushAll(ctx context.Context) error {
	cmd := r.client.FlushAll(ctx)
	return cmd.Err()
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
