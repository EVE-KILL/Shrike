package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func queryMaps(
	ctx context.Context,
	db Database,
	query string,
	args ...any,
) ([]map[string]any, error) {
	if db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToMap)
}

func firstMap(rows []map[string]any) map[string]any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

func valueString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	case fmt.Stringer:
		return value.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func nullableString(value any) *string {
	if value == nil {
		return nil
	}
	result := valueString(value)
	return &result
}

func valueInt64(value any) int64 {
	switch value := value.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case uint:
		return int64(value)
	case uint8:
		return int64(value)
	case uint16:
		return int64(value)
	case uint32:
		return int64(value)
	case uint64:
		if value > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(value)
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		result, _ := strconv.ParseInt(value, 10, 64)
		return result
	case []byte:
		result, _ := strconv.ParseInt(string(value), 10, 64)
		return result
	default:
		return 0
	}
}

func nullableInt64(value any) *int64 {
	if value == nil {
		return nil
	}
	result := valueInt64(value)
	return &result
}

func valueFloat64(value any) float64 {
	switch value := value.(type) {
	case float32:
		return float64(value)
	case float64:
		return value
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint64:
		return float64(value)
	case string:
		result, _ := strconv.ParseFloat(value, 64)
		return result
	case []byte:
		result, _ := strconv.ParseFloat(string(value), 64)
		return result
	default:
		return 0
	}
}

func nullableFloat64(value any) *float64 {
	if value == nil {
		return nil
	}
	result := valueFloat64(value)
	return &result
}

func valueBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int64:
		return value != 0
	case string:
		result, _ := strconv.ParseBool(value)
		return result
	default:
		return false
	}
}

func nullableTime(value any) *time.Time {
	switch value := value.(type) {
	case time.Time:
		return &value
	case *time.Time:
		return value
	case nil:
		return nil
	default:
		return nil
	}
}

func clamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
