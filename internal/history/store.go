package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Record 代表一条推荐历史记录
type Record struct {
	UserID    string `json:"user_id"`
	ItemName  string `json:"item_name"`
	Domain    string `json:"domain"` // e.g., "music", "movie"
	Timestamp int64  `json:"timestamp"`
}

// Store 定义历史记录存储接口
type Store interface {
	// GetRecentHistory 获取用户在指定 domain 下最近 N 天的推荐历史
	GetRecentHistory(userID string, domain string, days int) ([]string, error)
	// SaveHistory 保存推荐历史
	SaveHistory(userID string, domain string, items []string) error
}

// FileStore 基于文件的历史存储实现
type FileStore struct {
	filePath string
	mu       sync.RWMutex
	records  []Record // 内存缓存，用于快速查询
}

// NewFileStore 创建一个新的 FileStore
// 如果文件不存在，会自动创建
func NewFileStore(filePath string) (*FileStore, error) {
	fs := &FileStore{
		filePath: filePath,
		records:  make([]Record, 0),
	}

	if err := fs.load(); err != nil {
		return nil, err
	}

	return fs, nil
}

// load 从文件加载所有历史记录到内存
func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record Record
		if err := json.Unmarshal(line, &record); err != nil {
			// 忽略损坏的行，或者记录日志
			continue
		}
		s.records = append(s.records, record)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan history file: %w", err)
	}

	return nil
}

// GetRecentHistory 获取用户最近 N 天的历史记录 (返回 ItemName 列表)
func (s *FileStore) GetRecentHistory(userID string, domain string, days int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()
	cutoff := now - int64(days*24*60*60)
	
	var result []string
	// 简单的全量扫描，对于 MVP 来说性能足够
	// 如果数据量大，可以使用 map[userID]map[domain][]Record 的索引结构优化
	for _, r := range s.records {
		if r.UserID == userID && r.Domain == domain && r.Timestamp >= cutoff {
			result = append(result, r.ItemName)
		}
	}

	return result, nil
}

// SaveHistory 保存新的推荐历史到文件和内存
func (s *FileStore) SaveHistory(userID string, domain string, items []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file for appending: %w", err)
	}
	defer f.Close()

	now := time.Now().Unix()
	encoder := json.NewEncoder(f)

	for _, item := range items {
		record := Record{
			UserID:    userID,
			ItemName:  item,
			Domain:    domain,
			Timestamp: now,
		}

		// 1. 写入文件
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("failed to write history record: %w", err)
		}

		// 2. 更新内存
		s.records = append(s.records, record)
	}

	return nil
}
