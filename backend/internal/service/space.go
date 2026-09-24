package service

import (
	"errors"
	"fmt"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/gorm"
)

type SpaceService struct {
	db *gorm.DB
}

func NewSpaceService(db *gorm.DB) *SpaceService {
	return &SpaceService{db: db}
}

type CreateSpaceInput struct {
	Name        string `json:"name" binding:"required,min=2,max=128"`
	Description string `json:"description" binding:"max=512"`
	Prefix      string `json:"prefix" binding:"required,max=128"`
}

type UpdateSpaceInput struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=128"`
	Description string `json:"description" binding:"max=512"`
}

type UpdateMemberRoleInput struct {
	Role string `json:"role" binding:"required,oneof=space_admin member"`
}

func (s *SpaceService) Create(ownerID uint64, in CreateSpaceInput) (*model.Space, error) {
	var exists int64
	if err := s.db.Model(&model.Space{}).Where("name = ?", in.Name).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, fmt.Errorf("%w: space name already exists", ErrConflict)
	}
	prefix, err := model.NormalizePrefix(in.Prefix)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	var prefixExists int64
	if err := s.db.Model(&model.Space{}).Where("prefix = ?", prefix).Count(&prefixExists).Error; err != nil {
		return nil, err
	}
	if prefixExists > 0 {
		return nil, fmt.Errorf("%w: space prefix already exists", ErrConflict)
	}

	space := &model.Space{
		Name:        in.Name,
		Description: in.Description,
		Prefix:      prefix,
		OwnerID:     ownerID,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(space).Error; err != nil {
			return err
		}
		// applicant as space_admin
		if err := tx.Create(&model.SpaceMember{
			SpaceID: space.ID,
			UserID:  ownerID,
			Role:    model.RoleSpaceAdmin,
		}).Error; err != nil {
			return err
		}
		// add all system admins
		var admins []model.User
		if err := tx.Where("role = ?", model.RoleSystemAdmin).Find(&admins).Error; err != nil {
			return err
		}
		for _, admin := range admins {
			if admin.ID == ownerID {
				continue
			}
			if err := tx.Create(&model.SpaceMember{
				SpaceID: space.ID,
				UserID:  admin.ID,
				Role:    model.RoleSpaceAdmin,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return space, nil
}

func (s *SpaceService) List(userID uint64, isSystemAdmin bool) ([]model.Space, error) {
	var spaces []model.Space
	if isSystemAdmin {
		if err := s.db.Order("id desc").Find(&spaces).Error; err != nil {
			return nil, err
		}
		return spaces, nil
	}
	err := s.db.Joins("JOIN space_members ON space_members.space_id = spaces.id").
		Where("space_members.user_id = ?", userID).
		Order("spaces.id desc").
		Find(&spaces).Error
	return spaces, err
}

func (s *SpaceService) Get(id uint64) (*model.Space, error) {
	var space model.Space
	if err := s.db.First(&space, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &space, nil
}

func (s *SpaceService) Update(id uint64, in UpdateSpaceInput) (*model.Space, error) {
	space, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != "" && in.Name != space.Name {
		var count int64
		if err := s.db.Model(&model.Space{}).Where("name = ? AND id <> ?", in.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: space name already exists", ErrConflict)
		}
		updates["name"] = in.Name
	}
	if in.Description != space.Description {
		updates["description"] = in.Description
	}
	if len(updates) > 0 {
		if err := s.db.Model(space).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *SpaceService) Delete(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var groups []model.APIGroup
		if err := tx.Where("space_id = ?", id).Find(&groups).Error; err != nil {
			return err
		}
		for _, g := range groups {
			var apis []model.API
			if err := tx.Where("group_id = ?", g.ID).Find(&apis).Error; err != nil {
				return err
			}
			for _, a := range apis {
				if err := tx.Where("api_id = ?", a.ID).Delete(&model.APIVersion{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("group_id = ?", g.ID).Delete(&model.API{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("space_id = ?", id).Delete(&model.APIGroup{}).Error; err != nil {
			return err
		}
		var ups []model.Upstream
		if err := tx.Where("space_id = ?", id).Find(&ups).Error; err != nil {
			return err
		}
		for _, up := range ups {
			if err := tx.Where("upstream_id = ?", up.ID).Delete(&model.UpstreamTarget{}).Error; err != nil {
				return err
			}
			if err := tx.Where("upstream_id = ?", up.ID).Delete(&model.UpstreamGateway{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("space_id = ?", id).Delete(&model.Upstream{}).Error; err != nil {
			return err
		}
		if err := tx.Where("space_id = ?", id).Delete(&model.SpaceMember{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&model.Space{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *SpaceService) Join(spaceID, userID uint64) error {
	if _, err := s.Get(spaceID); err != nil {
		return err
	}
	var count int64
	if err := s.db.Model(&model.SpaceMember{}).Where("space_id = ? AND user_id = ?", spaceID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: already a member", ErrConflict)
	}
	return s.db.Create(&model.SpaceMember{
		SpaceID: spaceID,
		UserID:  userID,
		Role:    model.RoleMember,
	}).Error
}

func (s *SpaceService) ListMembers(spaceID uint64) ([]model.SpaceMember, error) {
	var members []model.SpaceMember
	err := s.db.Preload("User").Where("space_id = ?", spaceID).Find(&members).Error
	return members, err
}

func (s *SpaceService) UpdateMemberRole(spaceID, userID uint64, role string) error {
	res := s.db.Model(&model.SpaceMember{}).
		Where("space_id = ? AND user_id = ?", spaceID, userID).
		Update("role", role)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SpaceService) ListAvailable(userID uint64) ([]model.Space, error) {
	var spaces []model.Space
	err := s.db.Where("id NOT IN (?)",
		s.db.Model(&model.SpaceMember{}).Select("space_id").Where("user_id = ?", userID),
	).Order("id desc").Find(&spaces).Error
	return spaces, err
}
