package gocron

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// bounds provides a range of acceptable values (plus a map of name to value).
type bounds struct {
	min uint
	max uint
}

// The bounds for each field.
var (
	milliseconds = bounds{0, 999}
	seconds      = bounds{0, 59}
	minutes      = bounds{0, 59}
	hours        = bounds{0, 23}
	dom          = bounds{1, 31}
	months       = bounds{1, 12}
)

const (
	// Set the top bit if a star was included in the expression.
	starBit = 1 << 63
)

// 解析定时任务周期
func Parse(spec string) (Schedule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

	// 固定周期
	// Handle named schedules (descriptors), if configured
	const every = "@every "
	if strings.HasPrefix(spec, every) {
		duration, err := time.ParseDuration(spec[len(every):])
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration %s: %s", spec, err)
		}
		return Every(duration), nil
	}

	// 指定时间周期
	// Split on whitespace.
	fields := strings.Fields(spec)
	field := func(field string, r bounds) uint64 {
		var bits uint64
		bits, err := getField(field, r)
		if err != nil {
			return 0
		}
		return bits
	}

	s := &SpecSchedule{
		getMilliseconds(fields[5], milliseconds),
		field(fields[4], seconds),
		field(fields[3], minutes),
		field(fields[2], hours),
		field(fields[1], dom),
		field(fields[0], months),
	}
	return s, nil
}

// getField returns an Int with the bits set representing all of the times that
// the field represents or error parsing field value.  A "field" is a comma-separated
// list of "ranges".
func getField(field string, r bounds) (uint64, error) {
	var start, end uint
	var extra uint64

	if field == "*" {
		start = r.min
		end = r.max
		extra = uint64(starBit)
	} else {
		num, err := strconv.Atoi(field)
		if err != nil || start < 0 {
			return 0, fmt.Errorf("failed to parse int from %s: %s", field, err)
		}
		start = uint(num)
		end = start
		extra = 0
	}
	return getBits(start, end, 1) | extra, nil
}

// getBits sets all bits in the range [min, max], modulo the given step size.
func getBits(min, max, step uint) uint64 {
	return ^(math.MaxUint64 << (max + 1)) & (math.MaxUint64 << min)
}

func getMilliseconds(field string, r bounds) int64 {
	if field == "*" {
		return -1
	}
	num, err := strconv.Atoi(field)
	if err != nil || num > int(r.max) || num < int(r.min) {
		return -1
	}
	return int64(num)
}
