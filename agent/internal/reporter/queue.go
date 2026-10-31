package reporter

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"v2ray-dash/agent/internal/model"
)

// Queue retains unacknowledged snapshots across network failures and agent restarts.
type Queue struct {
	path    string
	pending []*model.NodeStatus
}

func OpenQueue(path string) (*Queue, error) {
	q := &Queue{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return q, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &q.pending); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *Queue) save() error {
	if err := os.MkdirAll(filepath.Dir(q.path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(q.pending)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(q.path), ".heartbeat-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, q.path)
}

func (q *Queue) Enqueue(status *model.NodeStatus) error {
	status.BatchID = uuid.NewString()
	q.pending = append(q.pending, status)
	return q.save()
}

func (q *Queue) Flush(send func(*model.NodeStatus) error) error {
	for len(q.pending) > 0 {
		if err := send(q.pending[0]); err != nil {
			return err
		}
		q.pending = q.pending[1:]
		if err := q.save(); err != nil {
			return err
		}
	}
	return nil
}
