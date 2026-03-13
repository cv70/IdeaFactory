package utils

import (
	"time"

	"github.com/sony/sonyflake"
)

var sonyflakeInstance *sonyflake.Sonyflake

func init() {    
	startTime := time.Unix(STARTS, 0)
	settings := sonyflake.Settings{
		StartTime: startTime,
	}
	sonyflakeInstance = sonyflake.NewSonyflake(settings)
}

func NewIDUint64() (uint64, error) {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{})
	return sf.NextID()
}

func NewIDInt64() (int64, error) {
	id, err := NewIDUint64()
	return int64(id), err
}

func IDUint64ToTime(id uint64) time.Time {
	return time.Unix(STARTS, int64(id<<22)).Add(time.Duration(id>>22) * time.Millisecond)
}

func IDInt64ToTime(id int64) time.Time {
	return IDUint64ToTime(uint64(id))
}
