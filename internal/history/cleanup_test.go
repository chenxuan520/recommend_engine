package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanup(t *testing.T) {
	// 1. 创建临时文件
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_history.jsonl")

	// 2. 准备数据：包含过期和未过期的数据
	now := time.Now().Unix()
	records := []Record{
		{UserID: "u1", ItemName: "old_item", Domain: "music", Timestamp: now - 8*24*3600}, // 8 days ago (expired)
		{UserID: "u1", ItemName: "new_item", Domain: "music", Timestamp: now - 1*24*3600}, // 1 day ago (kept)
		{UserID: "u2", ItemName: "just_expired", Domain: "video", Timestamp: now - 7*24*3600 - 100}, // > 7 days (expired)
		{UserID: "u2", ItemName: "just_kept", Domain: "video", Timestamp: now - 7*24*3600 + 100}, // < 7 days (kept)
	}

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	encoder := json.NewEncoder(f)
	for _, r := range records {
		if err := encoder.Encode(r); err != nil {
			t.Fatalf("failed to write record: %v", err)
		}
	}
	f.Close()

	// 3. 初始化 Store
	store, err := NewFileStore(filePath)
	if err != nil {
		t.Fatalf("failed to new file store: %v", err)
	}

	// 4. 执行清理 (保留 7 天)
	if err := store.Cleanup(7); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 5. 验证内存数据
	// 我们期望剩下 2 条记录
	expectedCount := 2
	// Access private field for testing? No, use GetRecentHistory or check file content.
	// But GetRecentHistory filters by user/domain/time.
	// Since we are in the same package `history`, we can access private fields `records`.
	if len(store.records) != expectedCount {
		t.Errorf("expected %d records, got %d", expectedCount, len(store.records))
	}

	for _, r := range store.records {
		if r.ItemName == "old_item" || r.ItemName == "just_expired" {
			t.Errorf("found expired item: %s", r.ItemName)
		}
	}

	// 6. 验证文件持久化
	// 重新加载 Store
	store2, err := NewFileStore(filePath)
	if err != nil {
		t.Fatalf("failed to reload file store: %v", err)
	}
	if len(store2.records) != expectedCount {
		t.Errorf("expected %d records after reload, got %d", expectedCount, len(store2.records))
	}
}
