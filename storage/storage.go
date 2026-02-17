package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	dataDirName    = "data"
	cardsFileName  = "cards.json"
	historyFileName = "history.json"
)

// CharacterCard 人物卡结构
type CharacterCard struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	System    string                 `json:"system"` // "coc7" or "dnd5e"
	PlayerID  int64                  `json:"player_id"`
	GroupID   int64                  `json:"group_id,omitempty"`
	Attrs     map[string]interface{} `json:"attrs"`
	Created   int64                  `json:"created"`
	Updated   int64                  `json:"updated"`
}

// RollHistory 掷骰历史
type RollHistory struct {
	ID         int64  `json:"id"`
	PlayerID   int64  `json:"player_id"`
	GroupID    int64  `json:"group_id,omitempty"`
	Expression string `json:"expression"`
	Result     string `json:"result"`
	Time       int64  `json:"time"`
}

// Storage 数据存储管理器
type Storage struct {
	dataDir     string
	cardsPath   string
	historyPath string
	mu          sync.RWMutex
	cards       map[string]*CharacterCard
	history     []RollHistory
}

// New 创建新的存储管理器
func New() (*Storage, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("无法获取工作目录: %v", err)
		wd = "."
	}

	dataDir := filepath.Join(wd, dataDirName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	cardsPath := filepath.Join(dataDir, cardsFileName)
	historyPath := filepath.Join(dataDir, historyFileName)

	s := &Storage{
		dataDir:     dataDir,
		cardsPath:   cardsPath,
		historyPath: historyPath,
		cards:       make(map[string]*CharacterCard),
		history:     make([]RollHistory, 0),
	}

	// 加载现有数据
	if err := s.loadCards(); err != nil {
		log.Printf("加载人物卡失败: %v", err)
	}
	if err := s.loadHistory(); err != nil {
		log.Printf("加载历史记录失败: %v", err)
	}

	return s, nil
}

// loadCards 加载人物卡数据
func (s *Storage) loadCards() error {
	if _, err := os.Stat(s.cardsPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.cardsPath)
	if err != nil {
		return err
	}

	var cards map[string]*CharacterCard
	if err := json.Unmarshal(data, &cards); err != nil {
		return err
	}

	s.cards = cards
	return nil
}

// saveCards 保存人物卡数据
func (s *Storage) saveCards() error {
	data, err := json.MarshalIndent(s.cards, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.cardsPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// loadHistory 加载历史记录
func (s *Storage) loadHistory() error {
	if _, err := os.Stat(s.historyPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(s.historyPath)
	if err != nil {
		return err
	}

	var history []RollHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	s.history = history
	return nil
}

// saveHistory 保存历史记录
func (s *Storage) saveHistory() error {
	// 只保留最近1000条记录
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}

	data, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.historyPath, data, 0644); err != nil {
		return err
	}

	return nil
}

// SaveCard 保存人物卡
func (s *Storage) SaveCard(card *CharacterCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	card.Updated = now
	if card.Created == 0 {
		card.Created = now
	}

	s.cards[card.ID] = card
	return s.saveCards()
}

// GetCard 获取人物卡
func (s *Storage) GetCard(id string) (*CharacterCard, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	card, ok := s.cards[id]
	return card, ok
}

// GetCardsByPlayer 获取玩家的所有人物卡
func (s *Storage) GetCardsByPlayer(playerID int64) []*CharacterCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*CharacterCard
	for _, card := range s.cards {
		if card.PlayerID == playerID {
			result = append(result, card)
		}
	}
	return result
}

// GetCardsByGroup 获取群组的所有人物卡
func (s *Storage) GetCardsByGroup(groupID int64) []*CharacterCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*CharacterCard
	for _, card := range s.cards {
		if card.GroupID == groupID {
			result = append(result, card)
		}
	}
	return result
}

// DeleteCard 删除人物卡
func (s *Storage) DeleteCard(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.cards[id]; !ok {
		return fmt.Errorf("人物卡不存在: %s", id)
	}

	delete(s.cards, id)
	return s.saveCards()
}

// AddHistory 添加掷骰历史
func (s *Storage) AddHistory(h *RollHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	h.Time = time.Now().Unix()
	s.history = append(s.history, *h)
	return s.saveHistory()
}

// GetHistory 获取掷骰历史
func (s *Storage) GetHistory(playerID int64, limit int) []RollHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	var result []RollHistory
	count := 0
	for i := len(s.history) - 1; i >= 0 && count < limit; i-- {
		if s.history[i].PlayerID == playerID {
			result = append(result, s.history[i])
			count++
		}
	}
	return result
}
