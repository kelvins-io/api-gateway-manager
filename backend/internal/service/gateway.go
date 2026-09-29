package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/gorm"
)

type GatewayService struct {
	db *gorm.DB
}

func NewGatewayService(db *gorm.DB) *GatewayService {
	return &GatewayService{db: db}
}

type CreateGatewayInput struct {
	Name        string `json:"name" binding:"required,min=2,max=128"`
	AdminAPI    string `json:"admin_api" binding:"required,min=8,max=512"`
	Domain      string `json:"domain" binding:"required,max=255"`
	NetworkZone string `json:"network_zone" binding:"required,oneof=内网 DMZ"`
	Shared      *bool  `json:"shared"`
}

type UpdateGatewayInput struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=128"`
	AdminAPI    string `json:"admin_api" binding:"omitempty,min=8,max=512"`
	Domain      string `json:"domain" binding:"omitempty,max=255"`
	NetworkZone string `json:"network_zone" binding:"omitempty,oneof=内网 DMZ"`
	Shared      *bool  `json:"shared"`
}

func normalizeGatewayDomain(raw string) (string, error) {
	domain := strings.TrimSpace(raw)
	host, portText, err := net.SplitHostPort(domain)
	if err != nil || host == "" {
		return "", fmt.Errorf("%w: domain must be ip:port or domain:port", ErrBadRequest)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("%w: domain port must be between 1 and 65535", ErrBadRequest)
	}
	if ip := net.ParseIP(host); ip != nil {
		return net.JoinHostPort(ip.String(), strconv.Itoa(port)), nil
	}
	if !validDomainHost(host) {
		return "", fmt.Errorf("%w: domain must be ip:port or domain:port", ErrBadRequest)
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func validDomainHost(host string) bool {
	if host == "" || len(host) > 253 || strings.Contains(host, "..") {
		return false
	}
	hasLetter := false
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
				hasLetter = true
			case r >= '0' && r <= '9', r == '-':
			default:
				if unicode.IsLetter(r) {
					hasLetter = true
					continue
				}
				return false
			}
		}
	}
	return hasLetter
}

func (s *GatewayService) probeAdminAPI(adminAPI string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := kongclient.Probe(ctx, adminAPI); err != nil {
		return fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	return nil
}

func (s *GatewayService) Probe(adminAPI string) error {
	return s.probeAdminAPI(adminAPI)
}

func (s *GatewayService) Create(in CreateGatewayInput) (*model.Gateway, error) {
	var count int64
	if err := s.db.Model(&model.Gateway{}).Where("name = ?", in.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: gateway name already exists", ErrConflict)
	}
	domain, err := normalizeGatewayDomain(in.Domain)
	if err != nil {
		return nil, err
	}
	if err := s.probeAdminAPI(in.AdminAPI); err != nil {
		return nil, err
	}
	shared := true
	if in.Shared != nil {
		shared = *in.Shared
	}
	gw := &model.Gateway{
		Name:        in.Name,
		AdminAPI:    in.AdminAPI,
		Domain:      domain,
		NetworkZone: in.NetworkZone,
		Shared:      shared,
	}
	if err := s.db.Select("Name", "AdminAPI", "Domain", "NetworkZone", "Shared").Create(gw).Error; err != nil {
		return nil, err
	}
	return gw, nil
}

func (s *GatewayService) List() ([]model.Gateway, error) {
	var list []model.Gateway
	err := s.db.Order("id desc").Find(&list).Error
	return list, err
}

type GatewayOption struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	NetworkZone string `json:"network_zone"`
}

func (s *GatewayService) ListOptions() ([]GatewayOption, error) {
	return s.listOptions(nil)
}

func (s *GatewayService) ListOptionsForSpace(spaceID uint64) ([]GatewayOption, error) {
	return s.listOptions(&spaceID)
}

func (s *GatewayService) listOptions(spaceID *uint64) ([]GatewayOption, error) {
	var list []model.Gateway
	q := s.db.Select("id", "name", "network_zone").Order("id desc")
	if spaceID == nil {
		q = q.Where("shared = ?", true)
	} else {
		q = q.Where(
			"shared = ? OR id IN (SELECT gateway_id FROM gateway_spaces WHERE space_id = ?)",
			true, *spaceID,
		)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	out := make([]GatewayOption, 0, len(list))
	for _, g := range list {
		out = append(out, GatewayOption{
			ID:          g.ID,
			Name:        g.Name,
			NetworkZone: g.NetworkZone,
		})
	}
	return out, nil
}

func (s *GatewayService) UsableBySpace(gatewayID, spaceID uint64) error {
	var gw model.Gateway
	if err := s.db.First(&gw, gatewayID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: gateway not found", ErrBadRequest)
		}
		return err
	}
	if gw.Shared {
		return nil
	}
	var count int64
	if err := s.db.Model(&model.GatewaySpace{}).
		Where("gateway_id = ? AND space_id = ?", gatewayID, spaceID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: gateway is not authorized for this space", ErrBadRequest)
	}
	return nil
}

func (s *GatewayService) ListUnauthorizedSpaces(gatewayID uint64) ([]model.Space, error) {
	gw, err := s.Get(gatewayID)
	if err != nil {
		return nil, err
	}
	if gw.Shared {
		return nil, fmt.Errorf("%w: shared gateway does not need authorization", ErrBadRequest)
	}
	var authorizedIDs []uint64
	if err := s.db.Model(&model.GatewaySpace{}).Where("gateway_id = ?", gatewayID).Pluck("space_id", &authorizedIDs).Error; err != nil {
		return nil, err
	}
	q := s.db.Where("status = ?", model.SpaceStatusActive).Order("id desc")
	if len(authorizedIDs) > 0 {
		q = q.Where("id NOT IN ?", authorizedIDs)
	}
	var list []model.Space
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

type AuthorizeSpacesInput struct {
	SpaceIDs []uint64 `json:"space_ids" binding:"required"`
}

func (s *GatewayService) AuthorizeSpaces(gatewayID uint64, spaceIDs []uint64) error {
	gw, err := s.Get(gatewayID)
	if err != nil {
		return err
	}
	if gw.Shared {
		return fmt.Errorf("%w: shared gateway does not need authorization", ErrBadRequest)
	}
	spaceIDs = uniqueIDs(spaceIDs)
	if len(spaceIDs) == 0 {
		return fmt.Errorf("%w: space_ids required", ErrBadRequest)
	}
	var spaces []model.Space
	if err := s.db.Where("id IN ? AND status = ?", spaceIDs, model.SpaceStatusActive).Find(&spaces).Error; err != nil {
		return err
	}
	if len(spaces) != len(spaceIDs) {
		return fmt.Errorf("%w: space not found or inactive", ErrBadRequest)
	}
	var existing []uint64
	if err := s.db.Model(&model.GatewaySpace{}).
		Where("gateway_id = ? AND space_id IN ?", gatewayID, spaceIDs).
		Pluck("space_id", &existing).Error; err != nil {
		return err
	}
	have := map[uint64]struct{}{}
	for _, id := range existing {
		have[id] = struct{}{}
	}
	rows := make([]model.GatewaySpace, 0, len(spaceIDs))
	for _, id := range spaceIDs {
		if _, ok := have[id]; ok {
			continue
		}
		rows = append(rows, model.GatewaySpace{GatewayID: gatewayID, SpaceID: id})
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.Create(&rows).Error
}

func (s *GatewayService) Get(id uint64) (*model.Gateway, error) {
	var gw model.Gateway
	if err := s.db.First(&gw, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &gw, nil
}

func (s *GatewayService) Update(id uint64, in UpdateGatewayInput) (*model.Gateway, error) {
	gw, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != "" {
		var count int64
		if err := s.db.Model(&model.Gateway{}).Where("name = ? AND id <> ?", in.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("%w: gateway name already exists", ErrConflict)
		}
		updates["name"] = in.Name
	}
	if in.AdminAPI != "" && in.AdminAPI != gw.AdminAPI {
		if err := s.probeAdminAPI(in.AdminAPI); err != nil {
			return nil, err
		}
		updates["admin_api"] = in.AdminAPI
	}
	if in.Domain != "" {
		domain, err := normalizeGatewayDomain(in.Domain)
		if err != nil {
			return nil, err
		}
		updates["domain"] = domain
	}
	if in.NetworkZone != "" {
		updates["network_zone"] = in.NetworkZone
	}
	if in.Shared != nil {
		updates["shared"] = *in.Shared
	}
	if len(updates) > 0 {
		if err := s.db.Model(gw).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *GatewayService) Delete(id uint64) error {
	var count int64
	if err := s.db.Model(&model.APIGroup{}).Where("gateway_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: gateway is used by api groups", ErrConflict)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("gateway_id = ?", id).Delete(&model.GatewaySpace{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&model.Gateway{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}
