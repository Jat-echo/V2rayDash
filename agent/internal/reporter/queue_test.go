package reporter

import (
	"errors"
	"path/filepath"
	"testing"
	"v2ray-dash/agent/internal/model"
)

func TestQueueRetainsOriginalBatchAcrossFailureAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	q, err := OpenQueue(path)
	if err != nil {
		t.Fatal(err)
	}
	status := &model.NodeStatus{ServerID: "node", TrafficEpoch: "core-1", UserTrafficStats: []model.UserTrafficStat{{Email: "user", Upload: 50, Download: 100}}}
	if err = q.Enqueue(status); err != nil {
		t.Fatal(err)
	}
	if err = q.Flush(func(*model.NodeStatus) error { return errors.New("offline") }); err == nil {
		t.Fatal("failure was hidden")
	}
	restarted, err := OpenQueue(path)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	err = restarted.Flush(func(saved *model.NodeStatus) error {
		count++
		if saved.BatchID != status.BatchID || saved.UserTrafficStats[0].Download != 100 {
			t.Fatal("pending snapshot changed")
		}
		return nil
	})
	if err != nil || count != 1 {
		t.Fatalf("flush=%d err=%v", count, err)
	}
	clean, err := OpenQueue(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = clean.Flush(func(*model.NodeStatus) error { t.Fatal("acknowledged batch was replayed"); return nil }); err != nil {
		t.Fatal(err)
	}
}
