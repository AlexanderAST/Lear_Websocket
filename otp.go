package main

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type OTP struct {
	Key     string
	Created time.Time
}

type RetentionMap map[string]OTP

func NewRetentionMap(ctx context.Context, retetionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)

	go rm.Retetion(ctx, retetionPeriod)

	return rm
}

func (rm RetentionMap) NewOTP() OTP {
	o := OTP{
		Key:     uuid.NewString(),
		Created: time.Now(),
	}
	rm[o.Key] = o
	return o
}

func (rm RetentionMap) VerifyOTP(otp string) bool {
	if _, ok := rm[otp]; !ok {
		return false
	}
	delete(rm, otp)
	return true
}

func (rm RetentionMap) Retetion(ctx context.Context, retetionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				if otp.Created.Add(retetionPeriod).Before(time.Now()) {
					delete(rm, otp.Key)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
