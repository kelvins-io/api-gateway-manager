package service

import (
	"errors"
	"fmt"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/gorm"
)

type GroupService struct {
	db *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

type CreateGroupInput struct {
	Name      string `json:"name" binding:"required,min=1,max=128"`
	GatewayID uint64 `json:"gateway_id" binding:"required"`
}

type UpdateGroupInput struct {
	Name string `json:"name" binding:"omitempty,min=1,max=128"`
}

func (s *GroupService) Create(spaceID uint64, in CreateGroupInput) (*model.APIGroup, error) {
	if err := NewGatewayService(s.db).UsableBySpace(in.GatewayID, spaceID); err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&model.APIGroup{}).Where("space_id = ? AND name = ?", spaceID, in.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: 分组名已存在", ErrConflict)
	}
	group := &model.APIGroup{
		SpaceID:   spaceID,
		GatewayID: in.GatewayID,
		Name:      in.Name,
	}
	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}
	return s.Get(group.ID)
}

func (s *GroupService) ListBySpace(spaceID uint64) ([]model.APIGroup, error) {
	var list []model.APIGroup
	if err := s.db.Preload("Gateway").Where("space_id = ?", spaceID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	if err := fillGroupAPICounts(s.db, list); err != nil {
		return nil, err
	}
	return list, nil
}

func fillGroupAPICounts(db *gorm.DB, list []model.APIGroup) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uint64, len(list))
	for i, g := range list {
		ids[i] = g.ID
	}
	type row struct {
		GroupID uint64
		Cnt     int64
	}
	var rows []row
	if err := db.Model(&model.API{}).Select("group_id, count(*) as cnt").Where("group_id IN ?", ids).Group("group_id").Scan(&rows).Error; err != nil {
		return err
	}
	counts := make(map[uint64]int64, len(rows))
	for _, r := range rows {
		counts[r.GroupID] = r.Cnt
	}
	for i := range list {
		list[i].APICount = counts[list[i].ID]
	}
	return nil
}

func (s *GroupService) Get(id uint64) (*model.APIGroup, error) {
	var group model.APIGroup
	if err := s.db.Preload("Gateway").First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &group, nil
}

func (s *GroupService) Update(id uint64, in UpdateGroupInput) (*model.APIGroup, error) {
	group, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != "" && in.Name != group.Name {
		var count int64
		if err := s.db.Model(&model.APIGroup{}).Where("space_id = ? AND name = ? AND id <> ?", group.SpaceID, in.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: 分组名已存在", ErrConflict)
		}
		updates["name"] = in.Name
	}
	if len(updates) > 0 {
		if err := s.db.Model(group).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *GroupService) Delete(id uint64) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	var apiCount int64
	if err := s.db.Model(&model.API{}).Where("group_id = ?", id).Count(&apiCount).Error; err != nil {
		return err
	}
	if apiCount > 0 {
		return fmt.Errorf("%w: 分组下仍有 API，不能删除", ErrConflict)
	}
	res := s.db.Delete(&model.APIGroup{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GroupService) SpaceIDOfGroup(groupID uint64) (uint64, error) {
	group, err := s.Get(groupID)
	if err != nil {
		return 0, err
	}
	return group.SpaceID, nil
}
