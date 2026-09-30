package service

import (
	"context"
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

type AddMemberInput struct {
	UserID uint64 `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"omitempty,oneof=space_admin member"`
}

func (s *SpaceService) Create(ownerID uint64, isSystemAdmin bool, in CreateSpaceInput) (*model.Space, error) {
	var exists int64
	if err := s.db.Model(&model.Space{}).Where("name = ?", in.Name).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, fmt.Errorf("%w: 空间名已存在", ErrConflict)
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

	status := model.SpaceStatusPending
	if isSystemAdmin {
		status = model.SpaceStatusActive
	}

	space := &model.Space{
		Name:        in.Name,
		Description: in.Description,
		Prefix:      prefix,
		OwnerID:     ownerID,
		Status:      status,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(space).Error; err != nil {
			return err
		}
		// applicant as space_admin (active membership so they can track the application)
		if err := tx.Create(&model.SpaceMember{
			SpaceID: space.ID,
			UserID:  ownerID,
			Role:    model.RoleSpaceAdmin,
			Status:  model.MemberStatusActive,
		}).Error; err != nil {
			return err
		}
		// system admins are added only when space becomes active
		if status == model.SpaceStatusActive {
			return ensureSystemAdminsInSpace(tx, space.ID, ownerID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return space, nil
}

func ensureSystemAdminsInSpace(tx *gorm.DB, spaceID, skipUserID uint64) error {
	var admins []model.User
	if err := tx.Where("role = ?", model.RoleSystemAdmin).Find(&admins).Error; err != nil {
		return err
	}
	for _, admin := range admins {
		if admin.ID == skipUserID {
			continue
		}
		var count int64
		if err := tx.Model(&model.SpaceMember{}).Where("space_id = ? AND user_id = ?", spaceID, admin.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := tx.Create(&model.SpaceMember{
			SpaceID: spaceID,
			UserID:  admin.ID,
			Role:    model.RoleSpaceAdmin,
			Status:  model.MemberStatusActive,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *SpaceService) List(userID uint64, isSystemAdmin bool) ([]model.Space, error) {
	var spaces []model.Space
	if isSystemAdmin {
		if err := s.db.Order("id desc").Find(&spaces).Error; err != nil {
			return nil, err
		}
		for i := range spaces {
			spaces[i].MemberStatus = model.MemberStatusActive
		}
		if err := fillSpaceResourceCounts(s.db, spaces); err != nil {
			return nil, err
		}
		return spaces, nil
	}

	type row struct {
		model.Space
		MemberStatus string `gorm:"column:member_status"`
	}
	var rows []row
	err := s.db.Table("spaces").
		Select("spaces.*, space_members.status AS member_status").
		Joins("JOIN space_members ON space_members.space_id = spaces.id").
		Where("space_members.user_id = ?", userID).
		Order("spaces.id desc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	spaces = make([]model.Space, len(rows))
	for i, r := range rows {
		spaces[i] = r.Space
		spaces[i].MemberStatus = r.MemberStatus
	}
	if err := fillSpaceResourceCounts(s.db, spaces); err != nil {
		return nil, err
	}
	return spaces, nil
}

func fillSpaceResourceCounts(db *gorm.DB, list []model.Space) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uint64, len(list))
	for i, space := range list {
		ids[i] = space.ID
	}
	type row struct {
		SpaceID uint64
		Cnt     int64
	}
	fill := func(dest interface{}, set func(int, int64)) error {
		var rows []row
		if err := db.Model(dest).Select("space_id, count(*) as cnt").Where("space_id IN ?", ids).Group("space_id").Scan(&rows).Error; err != nil {
			return err
		}
		counts := make(map[uint64]int64, len(rows))
		for _, r := range rows {
			counts[r.SpaceID] = r.Cnt
		}
		for i := range list {
			set(i, counts[list[i].ID])
		}
		return nil
	}
	if err := fill(&model.APIGroup{}, func(i int, n int64) { list[i].GroupCount = n }); err != nil {
		return err
	}
	if err := fill(&model.Upstream{}, func(i int, n int64) { list[i].UpstreamCount = n }); err != nil {
		return err
	}
	if err := fill(&model.Consumer{}, func(i int, n int64) { list[i].ConsumerCount = n }); err != nil {
		return err
	}
	return fill(&model.Plugin{}, func(i int, n int64) { list[i].PluginCount = n })
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
	if space.Status != model.SpaceStatusActive {
		return nil, fmt.Errorf("%w: space is not active", ErrBadRequest)
	}
	updates := map[string]interface{}{}
	if in.Name != "" && in.Name != space.Name {
		var count int64
		if err := s.db.Model(&model.Space{}).Where("name = ? AND id <> ?", in.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: 空间名已存在", ErrConflict)
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

func (s *SpaceService) Delete(_ context.Context, id uint64) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	checks := []struct {
		model interface{}
		msg   string
	}{
		{&model.APIGroup{}, "空间下仍有分组，不能删除"},
		{&model.Upstream{}, "空间下仍有 Upstream，不能删除"},
		{&model.Consumer{}, "空间下仍有 Consumer，不能删除"},
		{&model.Plugin{}, "空间下仍有 Plugin，不能删除"},
	}
	for _, c := range checks {
		var count int64
		if err := s.db.Model(c.model).Where("space_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%w: %s", ErrConflict, c.msg)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Use raw SQL so GORM zero-PK Delete quirks cannot leave child rows
		// that block the spaces foreign key.
		for _, stmt := range []string{
			"DELETE FROM gateway_spaces WHERE space_id = ?",
			"DELETE FROM space_members WHERE space_id = ?",
		} {
			if err := tx.Exec(stmt, id).Error; err != nil {
				return err
			}
		}

		res := tx.Exec("DELETE FROM spaces WHERE id = ?", id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *SpaceService) Approve(id uint64) (*model.Space, error) {
	space, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if space.Status == model.SpaceStatusActive {
		return space, nil
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(space).Update("status", model.SpaceStatusActive).Error; err != nil {
			return err
		}
		return ensureSystemAdminsInSpace(tx, id, space.OwnerID)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *SpaceService) Reject(ctx context.Context, id uint64) error {
	space, err := s.Get(id)
	if err != nil {
		return err
	}
	if space.Status != model.SpaceStatusPending {
		return fmt.Errorf("%w: only pending space can be rejected", ErrBadRequest)
	}
	return s.Delete(ctx, id)
}

func (s *SpaceService) Join(spaceID uint64, userID uint64, isSystemAdmin bool) error {
	space, err := s.Get(spaceID)
	if err != nil {
		return err
	}
	if space.Status != model.SpaceStatusActive {
		return fmt.Errorf("%w: space is not active", ErrBadRequest)
	}
	var count int64
	if err := s.db.Model(&model.SpaceMember{}).Where("space_id = ? AND user_id = ?", spaceID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: already a member or pending", ErrConflict)
	}
	status := model.MemberStatusPending
	role := model.RoleMember
	if isSystemAdmin {
		status = model.MemberStatusActive
		role = model.RoleSpaceAdmin
	}
	return s.db.Create(&model.SpaceMember{
		SpaceID: spaceID,
		UserID:  userID,
		Role:    role,
		Status:  status,
	}).Error
}

func (s *SpaceService) AddMember(spaceID uint64, in AddMemberInput) error {
	space, err := s.Get(spaceID)
	if err != nil {
		return err
	}
	if space.Status != model.SpaceStatusActive {
		return fmt.Errorf("%w: space is not active", ErrBadRequest)
	}
	var user model.User
	if err := s.db.First(&user, in.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: user not found", ErrNotFound)
		}
		return err
	}
	var existing model.SpaceMember
	err = s.db.Where("space_id = ? AND user_id = ?", spaceID, in.UserID).First(&existing).Error
	if err == nil {
		if existing.Status == model.MemberStatusPending {
			role := in.Role
			if role == "" {
				role = model.RoleMember
			}
			return s.db.Model(&existing).Updates(map[string]interface{}{
				"status": model.MemberStatusActive,
				"role":   role,
			}).Error
		}
		return fmt.Errorf("%w: already a member", ErrConflict)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	role := in.Role
	if role == "" {
		role = model.RoleMember
	}
	return s.db.Create(&model.SpaceMember{
		SpaceID: spaceID,
		UserID:  in.UserID,
		Role:    role,
		Status:  model.MemberStatusActive,
	}).Error
}

func (s *SpaceService) ApproveMember(spaceID, userID uint64) error {
	space, err := s.Get(spaceID)
	if err != nil {
		return err
	}
	if space.Status != model.SpaceStatusActive {
		return fmt.Errorf("%w: space is not active", ErrBadRequest)
	}
	var member model.SpaceMember
	if err := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if member.Status == model.MemberStatusActive {
		return nil
	}
	return s.db.Model(&member).Update("status", model.MemberStatusActive).Error
}

func (s *SpaceService) RejectMember(spaceID, userID uint64) error {
	var member model.SpaceMember
	if err := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if member.Status != model.MemberStatusPending {
		return fmt.Errorf("%w: only pending member can be rejected", ErrBadRequest)
	}
	return s.db.Delete(&member).Error
}

func (s *SpaceService) RemoveMember(spaceID uint64, userID uint64) error {
	space, err := s.Get(spaceID)
	if err != nil {
		return err
	}
	if space.OwnerID == userID {
		return fmt.Errorf("%w: cannot remove space owner", ErrBadRequest)
	}
	res := s.db.Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&model.SpaceMember{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SpaceService) ListMembers(spaceID uint64) ([]model.SpaceMember, error) {
	var members []model.SpaceMember
	err := s.db.Preload("User").Where("space_id = ?", spaceID).Order("id asc").Find(&members).Error
	return members, err
}

func (s *SpaceService) ListCandidateUsers(spaceID uint64) ([]model.User, error) {
	if _, err := s.Get(spaceID); err != nil {
		return nil, err
	}
	var users []model.User
	err := s.db.Where("id NOT IN (?)",
		s.db.Model(&model.SpaceMember{}).Select("user_id").Where("space_id = ?", spaceID),
	).Order("id asc").Find(&users).Error
	return users, err
}

func (s *SpaceService) UpdateMemberRole(spaceID uint64, userID uint64, role string) error {
	res := s.db.Model(&model.SpaceMember{}).
		Where("space_id = ? AND user_id = ? AND status = ?", spaceID, userID, model.MemberStatusActive).
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
	err := s.db.Where("status = ? AND id NOT IN (?)",
		model.SpaceStatusActive,
		s.db.Model(&model.SpaceMember{}).Select("space_id").Where("user_id = ?", userID),
	).Order("id desc").Find(&spaces).Error
	return spaces, err
}
