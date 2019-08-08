package main

import (
	"context"
	pb "git.ddex.io/infrastructure/ethereum-launcher/messages"
	"git.ddex.io/lib/monitor"
	"time"
)

func startDatabaseExporter(ctx context.Context) {
	pendingStatusName := pb.LaunchLogStatus_name[int32(pb.LaunchLogStatus_PENDING)]

	for {
		time.Sleep(10 * time.Second)

		launchLogs := getAllLogsWithStatus(pendingStatusName)
		monitor.Value("pending_log", float64(len(launchLogs)))

		longPendingLogCount := 0
		for _, l := range launchLogs {
			if l.CreatedAt.Before(time.Now().Add(-1 * 10 * time.Minute)) {
				longPendingLogCount += 1
			}
		}

		monitor.Value("long_pending_log", float64(longPendingLogCount))
	}
}