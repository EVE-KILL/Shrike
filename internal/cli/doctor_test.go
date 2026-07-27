package cli

import "testing"

func TestRedisVersionPrefersValkeyVersion(t *testing.T) {
	info := "# Server\r\nredis_version:7.2.4\r\nvalkey_version:8.1.9\r\n"
	if got := redisVersion(info); got != "Valkey 8.1.9" {
		t.Errorf("redisVersion = %q, want Valkey 8.1.9", got)
	}
}

func TestRedisVersionReportsRedis(t *testing.T) {
	info := "# Server\r\nredis_version:7.4.2\r\n"
	if got := redisVersion(info); got != "Redis 7.4.2" {
		t.Errorf("redisVersion = %q, want Redis 7.4.2", got)
	}
}
