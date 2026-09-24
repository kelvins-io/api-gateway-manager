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
}

type UpdateGatewayInput struct {
	Name        string `json:"name" binding:"omitempty,min=2,max=128"`
	Domain      string `json:"domain" binding:"omitempty,max=255"`
	NetworkZone string `json:"network_zone" binding:"omitempty,oneof=内网 DMZ"`
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
	gw := &model.Gateway{
		Name:        in.Name,
		AdminAPI:    in.AdminAPI,
		Domain:      domain,
		NetworkZone: in.NetworkZone,
	}
	if err := s.db.Create(gw).Error; err != nil {
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
	var list []model.Gateway
	if err := s.db.Select("id", "name", "network_zone").Order("id desc").Find(&list).Error; err != nil {
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
	res := s.db.Delete(&model.Gateway{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
