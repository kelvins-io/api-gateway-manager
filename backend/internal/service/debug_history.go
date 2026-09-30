package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DebugHistoryService struct {
	db *gorm.DB
}

func NewDebugHistoryService(db *gorm.DB) *DebugHistoryService {
	return &DebugHistoryService{db: db}
}

type CreateDebugHistoryInput struct {
	Title   string                 `json:"title"`
	APIID   *uint64                `json:"api_id"`
	APIName string                 `json:"api_name"`
	Method  string                 `json:"method" binding:"required"`
	URL     string                 `json:"url" binding:"required"`
	Request map[string]interface{} `json:"request" binding:"required"`
}

func (s *DebugHistoryService) Create(spaceID, userID uint64, in CreateDebugHistoryInput) (*model.DebugHistory, error) {
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if method == "" {
		return nil, fmt.Errorf("%w: method is required", ErrBadRequest)
	}
	rawURL := strings.TrimSpace(in.URL)
	if rawURL == "" {
		return nil, fmt.Errorf("%w: url is required", ErrBadRequest)
	}
	if in.Request == nil {
		return nil, fmt.Errorf("%w: request is required", ErrBadRequest)
	}
	reqJSON, err := json.Marshal(in.Request)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid request snapshot", ErrBadRequest)
	}
	if len(reqJSON) > 512*1024 {
		return nil, fmt.Errorf("%w: request snapshot too large", ErrBadRequest)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = method + " " + rawURL
	}
	title = truncateRunes(title, 255)
	apiName := truncateRunes(strings.TrimSpace(in.APIName), 128)

	var apiID *uint64
	if in.APIID != nil && *in.APIID > 0 {
		apiID = in.APIID
	}

	item := &model.DebugHistory{
		SpaceID:   spaceID,
		CreatedBy: userID,
		APIID:     apiID,
		APIName:   apiName,
		Title:     title,
		Method:    method,
		URL:       truncateRunes(rawURL, 2048),
		Request:   datatypes.JSON(reqJSON),
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}
	if err := s.db.Preload("Creator").First(item, item.ID).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DebugHistoryService) List(spaceID uint64) ([]model.DebugHistory, error) {
	var list []model.DebugHistory
	err := s.db.Preload("Creator").
		Where("space_id = ?", spaceID).
		Order("id desc").
		Limit(200).
		Find(&list).Error
	return list, err
}

func (s *DebugHistoryService) Get(spaceID, id uint64) (*model.DebugHistory, error) {
	var item model.DebugHistory
	if err := s.db.Preload("Creator").Where("space_id = ? AND id = ?", spaceID, id).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: debug history not found", ErrNotFound)
		}
		return nil, err
	}
	return &item, nil
}

func (s *DebugHistoryService) Delete(spaceID, id, userID uint64, isSpaceAdmin bool) error {
	item, err := s.Get(spaceID, id)
	if err != nil {
		return err
	}
	if !isSpaceAdmin && item.CreatedBy != userID {
		return fmt.Errorf("%w: only creator or space admin can delete", ErrForbidden)
	}
	res := s.db.Where("space_id = ? AND id = ?", spaceID, id).Delete(&model.DebugHistory{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: debug history not found", ErrNotFound)
	}
	return nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
