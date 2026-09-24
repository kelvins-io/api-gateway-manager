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
	Name      string `json:"name" binding:"omitempty,min=1,max=128"`
	GatewayID uint64 `json:"gateway_id"`
}

func (s *GroupService) Create(spaceID uint64, in CreateGroupInput) (*model.APIGroup, error) {
	var gw model.Gateway
	if err := s.db.First(&gw, in.GatewayID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: gateway not found", ErrBadRequest)
		}
		return nil, err
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
	err := s.db.Preload("Gateway").Where("space_id = ?", spaceID).Order("id desc").Find(&list).Error
	return list, err
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
	if in.Name != "" {
		updates["name"] = in.Name
	}
	if in.GatewayID > 0 {
		var gw model.Gateway
		if err := s.db.First(&gw, in.GatewayID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("%w: gateway not found", ErrBadRequest)
			}
			return nil, err
		}
		updates["gateway_id"] = in.GatewayID
	}
	if len(updates) > 0 {
		if err := s.db.Model(group).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *GroupService) Delete(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var apis []model.API
		if err := tx.Where("group_id = ?", id).Find(&apis).Error; err != nil {
			return err
		}
		for _, a := range apis {
			if a.Status == model.APIStatusPublished {
				return fmt.Errorf("%w: please offline published apis first", ErrConflict)
			}
			if err := tx.Where("api_id = ?", a.ID).Delete(&model.APIVersion{}).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM api_plugins WHERE api_id = ?", a.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec("DELETE FROM api_consumers WHERE api_id = ?", a.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("group_id = ?", id).Delete(&model.API{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&model.APIGroup{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *GroupService) SpaceIDOfGroup(groupID uint64) (uint64, error) {
	group, err := s.Get(groupID)
	if err != nil {
		return 0, err
	}
	return group.SpaceID, nil
}
