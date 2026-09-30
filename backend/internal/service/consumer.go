package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var consumerNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)
var managedACLGroup = regexp.MustCompile(`^G-([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|\d+)$`)

type ConsumerService struct {
	db *gorm.DB
}

func NewConsumerService(db *gorm.DB) *ConsumerService {
	return &ConsumerService{db: db}
}

type CredentialInput struct {
	Plugin string            `json:"plugin"`
	Config map[string]string `json:"config"`
}

type UpsertConsumerInput struct {
	Username    string            `json:"username" binding:"required"`
	CustomID    string            `json:"custom_id"`
	Credentials []CredentialInput `json:"credentials"`
	APIIDs      []uint64          `json:"api_ids"`
}

func (s *ConsumerService) EnsureSpace(id uint64, spaceID uint64) error {
	var count int64
	if err := s.db.Model(&model.Consumer{}).Where("id = ? AND space_id = ?", id, spaceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ConsumerService) List(spaceID uint64) ([]model.Consumer, error) {
	var list []model.Consumer
	err := s.db.Preload("Credentials").
		Preload("APIs.Group.Gateway").
		Preload("APIs.Group.Space").
		Where("space_id = ?", spaceID).
		Order("id desc").
		Find(&list).Error
	return list, err
}

func (s *ConsumerService) Create(ctx context.Context, spaceID uint64, in UpsertConsumerInput) (*model.Consumer, error) {
	consumer, creds, err := buildConsumer(spaceID, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUnique(spaceID, consumer.Username, consumer.CustomID, 0); err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(consumer).Error; err != nil {
			return err
		}
		for i := range creds {
			creds[i].ConsumerID = consumer.ID
		}
		if len(creds) > 0 {
			return tx.Create(&creds).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	saved, err := s.Get(consumer.ID)
	if err != nil {
		return nil, err
	}
	if err := s.replaceAPIs(saved.ID, spaceID, in.APIIDs); err != nil {
		return nil, err
	}
	saved, err = s.Get(saved.ID)
	if err != nil {
		return nil, err
	}
	if err := s.syncOne(ctx, saved); err != nil {
		return nil, err
	}
	return s.Get(saved.ID)
}

func (s *ConsumerService) Get(id uint64) (*model.Consumer, error) {
	var consumer model.Consumer
	if err := s.db.Preload("Credentials").First(&consumer, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &consumer, nil
}

func (s *ConsumerService) Update(ctx context.Context, id uint64, in UpsertConsumerInput) (*model.Consumer, error) {
	current, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	consumer, creds, err := buildConsumer(current.SpaceID, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUnique(current.SpaceID, consumer.Username, consumer.CustomID, id); err != nil {
		return nil, err
	}
	if err := validateCredentialUpdate(current.Credentials, creds); err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(current).Updates(map[string]interface{}{
			"username":  consumer.Username,
			"custom_id": consumer.CustomID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("consumer_id = ?", id).Delete(&model.ConsumerCredential{}).Error; err != nil {
			return err
		}
		for i := range creds {
			creds[i].ConsumerID = id
		}
		if len(creds) > 0 {
			return tx.Create(&creds).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	previous, err := s.linkedAPIGateways(id)
	if err != nil {
		return nil, err
	}
	if err := s.replaceAPIs(id, current.SpaceID, in.APIIDs); err != nil {
		return nil, err
	}
	saved, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.syncLinkedUpdate(ctx, saved, previous); err != nil {
		return nil, err
	}
	if err := s.resyncBoundAPIs(ctx, id); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *ConsumerService) replaceAPIs(consumerID uint64, spaceID uint64, ids []uint64) error {
	consumer, err := s.Get(consumerID)
	if err != nil {
		return err
	}
	var current []model.API
	if err := s.db.Model(consumer).Association("APIs").Find(&current); err != nil {
		return err
	}
	ids = uniqueIDs(ids)
	apis := make([]model.API, 0, len(ids))
	if len(ids) > 0 {
		if err := s.db.Joins("JOIN api_groups ON api_groups.id = apis.group_id").
			Where("apis.id IN ? AND api_groups.space_id = ?", ids, spaceID).
			Find(&apis).Error; err != nil {
			return err
		}
		if len(apis) != len(ids) {
			return fmt.Errorf("%w: api not found in this space", ErrBadRequest)
		}
		for _, api := range apis {
			if !api.AuthEnabled {
				return fmt.Errorf("%w: api %s has auth disabled", ErrBadRequest, api.Name)
			}
			if !consumerHasPlugin(*consumer, api.AuthPlugin) {
				return fmt.Errorf("%w: consumer has no %s credential for api %s", ErrBadRequest, api.AuthPlugin, api.Name)
			}
		}
	}
	if err := s.db.Model(consumer).Association("APIs").Replace(apis); err != nil {
		return err
	}
	next := map[uint64]struct{}{}
	for _, id := range ids {
		next[id] = struct{}{}
	}
	for _, api := range current {
		if _, ok := next[api.ID]; ok {
			continue
		}
		if err := s.writeACLGroup(consumerID, model.APIACLGroup(api.ID), false); err != nil {
			return err
		}
	}
	for _, api := range apis {
		if err := s.writeACLGroup(consumerID, model.APIACLGroup(api.ID), true); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerService) setACLGroup(ctx context.Context, consumerID uint64, group string, add bool) error {
	if err := s.writeACLGroup(consumerID, group, add); err != nil {
		return err
	}
	saved, err := s.Get(consumerID)
	if err != nil {
		return err
	}
	return s.syncOne(ctx, saved)
}

func (s *ConsumerService) writeACLGroup(consumerID uint64, group string, add bool) error {
	consumer, err := s.Get(consumerID)
	if err != nil {
		return err
	}
	kept := make([]model.ConsumerCredential, 0, len(consumer.Credentials)+1)
	found := false
	for _, cred := range consumer.Credentials {
		if cred.Plugin == "acl" && credentialGroup(cred) == group {
			found = true
			if !add {
				continue
			}
		}
		kept = append(kept, model.ConsumerCredential{ConsumerID: consumerID, Plugin: cred.Plugin, Config: cred.Config})
	}
	if add && !found {
		raw, err := json.Marshal(map[string]string{"group": group})
		if err != nil {
			return err
		}
		kept = append(kept, model.ConsumerCredential{ConsumerID: consumerID, Plugin: "acl", Config: datatypes.JSON(raw)})
	}
	return s.replaceCredentials(consumerID, kept)
}

func (s *ConsumerService) ensureLinkedACLGroups(consumerID uint64) error {
	var apiIDs []uint64
	if err := s.db.Table("api_consumers").Where("consumer_id = ?", consumerID).Pluck("api_id", &apiIDs).Error; err != nil {
		return err
	}
	consumer, err := s.Get(consumerID)
	if err != nil {
		return err
	}
	kept := make([]model.ConsumerCredential, 0, len(consumer.Credentials)+len(apiIDs))
	present := map[string]struct{}{}
	for _, cred := range consumer.Credentials {
		kept = append(kept, model.ConsumerCredential{ConsumerID: consumerID, Plugin: cred.Plugin, Config: cred.Config})
		if cred.Plugin == "acl" {
			present[credentialGroup(cred)] = struct{}{}
		}
	}
	changed := false
	for _, apiID := range apiIDs {
		group := model.APIACLGroup(apiID)
		if _, ok := present[group]; ok {
			continue
		}
		raw, err := json.Marshal(map[string]string{"group": group})
		if err != nil {
			return err
		}
		kept = append(kept, model.ConsumerCredential{ConsumerID: consumerID, Plugin: "acl", Config: datatypes.JSON(raw)})
		changed = true
	}
	if !changed {
		return nil
	}
	return s.replaceCredentials(consumerID, kept)
}

func (s *ConsumerService) replaceCredentials(consumerID uint64, creds []model.ConsumerCredential) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("consumer_id = ?", consumerID).Delete(&model.ConsumerCredential{}).Error; err != nil {
			return err
		}
		if len(creds) == 0 {
			return nil
		}
		return tx.Create(&creds).Error
	})
}

func validateCredentialUpdate(current, next []model.ConsumerCredential) error {
	current = visibleCredentials(current)
	next = visibleCredentials(next)
	if len(current) != len(next) {
		return fmt.Errorf("%w: credentials cannot be added or removed", ErrBadRequest)
	}
	for i := range current {
		if current[i].Plugin != next[i].Plugin {
			return fmt.Errorf("%w: credential type cannot be changed", ErrBadRequest)
		}
	}
	return nil
}

func visibleCredentials(creds []model.ConsumerCredential) []model.ConsumerCredential {
	out := make([]model.ConsumerCredential, 0, len(creds))
	for _, cred := range creds {
		if cred.Plugin == "acl" && managedACLGroup.MatchString(credentialGroup(cred)) {
			continue
		}
		out = append(out, cred)
	}
	return out
}

func credentialGroup(cred model.ConsumerCredential) string {
	cfg := map[string]string{}
	_ = json.Unmarshal(cred.Config, &cfg)
	return cfg["group"]
}

func (s *ConsumerService) Delete(ctx context.Context, id uint64) error {
	consumer, err := s.Get(id)
	if err != nil {
		return err
	}
	var boundAPIs []model.API
	if err := s.db.Joins("JOIN api_consumers ON api_consumers.api_id = apis.id").
		Where("api_consumers.consumer_id = ? AND apis.status = ?", id, model.APIStatusPublished).
		Find(&boundAPIs).Error; err != nil {
		return err
	}
	var bindings []model.ConsumerGateway
	if err := s.db.Where("consumer_id = ?", id).Find(&bindings).Error; err != nil {
		return err
	}
	for _, b := range bindings {
		var gw model.Gateway
		if err := s.db.First(&gw, b.GatewayID).Error; err != nil {
			continue
		}
		client, err := kongclient.New(gw.AdminAPI)
		if err != nil {
			return err
		}
		if err := client.DeleteConsumer(ctx, b.KongConsumerID); err != nil {
			return err
		}
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM api_consumers WHERE consumer_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Where("consumer_id = ?", id).Delete(&model.ConsumerGateway{}).Error; err != nil {
			return err
		}
		if err := tx.Where("consumer_id = ?", id).Delete(&model.ConsumerCredential{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Consumer{}, consumer.ID).Error
	})
	if err != nil {
		return err
	}
	for _, api := range boundAPIs {
		full, err := (&APIService{db: s.db}).Get(api.ID)
		if err != nil {
			return err
		}
		if err := applyAPIPlugins(ctx, full); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerService) resyncBoundAPIs(ctx context.Context, consumerID uint64) error {
	var apis []model.API
	if err := s.db.Joins("JOIN api_consumers ON api_consumers.api_id = apis.id").
		Where("api_consumers.consumer_id = ? AND apis.status = ?", consumerID, model.APIStatusPublished).
		Preload("Group.Gateway").Preload("Plugins").Preload("Consumers.Credentials").
		Find(&apis).Error; err != nil {
		return err
	}
	for i := range apis {
		if err := applyAPIPlugins(ctx, &apis[i]); err != nil {
			return err
		}
	}
	return nil
}

// ReconcileSpace pushes every consumer in the space onto the gateways currently needed
// (space group gateways plus gateways of linked published APIs).
func (s *ConsumerService) ReconcileSpace(ctx context.Context, spaceID uint64) error {
	var consumers []model.Consumer
	if err := s.db.Preload("Credentials").Where("space_id = ?", spaceID).Find(&consumers).Error; err != nil {
		return err
	}
	for i := range consumers {
		if err := s.syncOne(ctx, &consumers[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerService) syncOne(ctx context.Context, consumer *model.Consumer) error {
	wanted, err := s.consumerWantedGateways(consumer.ID, consumer.SpaceID)
	if err != nil {
		return err
	}
	return s.syncToGateways(ctx, consumer, wanted)
}

// syncLinkedUpdate pushes a consumer update only to gateways of APIs linked to it.
// A gateway that no longer has any linked API is removed. Bindings on other gateways stay as they are.
func (s *ConsumerService) syncLinkedUpdate(ctx context.Context, consumer *model.Consumer, previous map[uint64]model.Gateway) error {
	wanted, err := s.linkedAPIGateways(consumer.ID)
	if err != nil {
		return err
	}
	if err := s.upsertGateways(ctx, consumer, wanted); err != nil {
		return err
	}
	for id, gw := range previous {
		if _, ok := wanted[id]; ok {
			continue
		}
		if err := s.deleteConsumerOnGateway(ctx, consumer.ID, gw); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerService) linkedAPIGateways(consumerID uint64) (map[uint64]model.Gateway, error) {
	wanted := map[uint64]model.Gateway{}
	var apis []model.API
	if err := s.db.Joins("JOIN api_consumers ON api_consumers.api_id = apis.id").
		Where("api_consumers.consumer_id = ?", consumerID).
		Preload("Group.Gateway").
		Find(&apis).Error; err != nil {
		return nil, err
	}
	for _, api := range apis {
		if api.Group == nil || api.Group.Gateway == nil {
			continue
		}
		wanted[api.Group.Gateway.ID] = *api.Group.Gateway
	}
	return wanted, nil
}

func (s *ConsumerService) consumerWantedGateways(consumerID, spaceID uint64) (map[uint64]model.Gateway, error) {
	wanted := map[uint64]model.Gateway{}
	spaceGWs, err := s.spaceGateways(spaceID)
	if err != nil {
		return nil, err
	}
	for _, gw := range spaceGWs {
		wanted[gw.ID] = gw
	}
	var apis []model.API
	if err := s.db.Joins("JOIN api_consumers ON api_consumers.api_id = apis.id").
		Where("api_consumers.consumer_id = ? AND apis.status = ?", consumerID, model.APIStatusPublished).
		Preload("Group.Gateway").
		Find(&apis).Error; err != nil {
		return nil, err
	}
	for _, api := range apis {
		if api.Group == nil || api.Group.Gateway == nil {
			continue
		}
		wanted[api.Group.Gateway.ID] = *api.Group.Gateway
	}
	return wanted, nil
}

func (s *ConsumerService) spaceGateways(spaceID uint64) ([]model.Gateway, error) {
	var groups []model.APIGroup
	if err := s.db.Where("space_id = ?", spaceID).Find(&groups).Error; err != nil {
		return nil, err
	}
	seen := map[uint64]struct{}{}
	ids := make([]uint64, 0, len(groups))
	for _, g := range groups {
		if _, ok := seen[g.GatewayID]; ok {
			continue
		}
		seen[g.GatewayID] = struct{}{}
		ids = append(ids, g.GatewayID)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var gateways []model.Gateway
	if err := s.db.Where("id IN ?", ids).Find(&gateways).Error; err != nil {
		return nil, err
	}
	return gateways, nil
}

func (s *ConsumerService) syncToGateways(ctx context.Context, consumer *model.Consumer, wanted map[uint64]model.Gateway) error {
	if err := s.upsertGateways(ctx, consumer, wanted); err != nil {
		return err
	}
	var bindings []model.ConsumerGateway
	if err := s.db.Where("consumer_id = ?", consumer.ID).Find(&bindings).Error; err != nil {
		return err
	}
	for _, b := range bindings {
		if _, ok := wanted[b.GatewayID]; ok {
			continue
		}
		var gw model.Gateway
		if err := s.db.First(&gw, b.GatewayID).Error; err == nil {
			client, err := kongclient.New(gw.AdminAPI)
			if err != nil {
				return err
			}
			if err := client.DeleteConsumer(ctx, b.KongConsumerID); err != nil {
				return err
			}
		}
		if err := s.db.Delete(&model.ConsumerGateway{}, b.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ConsumerService) upsertGateways(ctx context.Context, consumer *model.Consumer, wanted map[uint64]model.Gateway) error {
	var bindings []model.ConsumerGateway
	if err := s.db.Where("consumer_id = ?", consumer.ID).Find(&bindings).Error; err != nil {
		return err
	}
	bound := map[uint64]model.ConsumerGateway{}
	for _, b := range bindings {
		bound[b.GatewayID] = b
	}
	for id, gw := range wanted {
		client, err := kongclient.New(gw.AdminAPI)
		if err != nil {
			return err
		}
		existing := bound[id].KongConsumerID
		kongID, err := client.SyncConsumer(ctx, consumerToSync(*consumer, existing))
		if err != nil {
			return err
		}
		if existing == "" {
			row := model.ConsumerGateway{ConsumerID: consumer.ID, GatewayID: id, KongConsumerID: kongID}
			if err := s.db.Create(&row).Error; err != nil {
				return err
			}
		} else if kongID != existing {
			if err := s.db.Model(&model.ConsumerGateway{}).Where("id = ?", bound[id].ID).Update("kong_consumer_id", kongID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ConsumerService) deleteConsumerOnGateway(ctx context.Context, consumerID uint64, gw model.Gateway) error {
	var binding model.ConsumerGateway
	err := s.db.Where("consumer_id = ? AND gateway_id = ?", consumerID, gw.ID).First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	client, err := kongclient.New(gw.AdminAPI)
	if err != nil {
		return err
	}
	if err := client.DeleteConsumer(ctx, binding.KongConsumerID); err != nil {
		return err
	}
	return s.db.Delete(&model.ConsumerGateway{}, binding.ID).Error
}

func (s *ConsumerService) ensureUnique(spaceID uint64, username, customID string, excludeID uint64) error {
	var count int64
	q := s.db.Model(&model.Consumer{}).Where("space_id = ? AND username = ?", spaceID, username)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: consumer username already exists", ErrConflict)
	}
	if customID == "" {
		return nil
	}
	q = s.db.Model(&model.Consumer{}).Where("space_id = ? AND custom_id = ?", spaceID, customID)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: consumer custom_id already exists", ErrConflict)
	}
	return nil
}

func buildConsumer(spaceID uint64, in UpsertConsumerInput) (*model.Consumer, []model.ConsumerCredential, error) {
	username := strings.TrimSpace(in.Username)
	if !consumerNamePattern.MatchString(username) {
		return nil, nil, fmt.Errorf("%w: invalid consumer username", ErrBadRequest)
	}
	customID := strings.TrimSpace(in.CustomID)
	if customID != "" && !consumerNamePattern.MatchString(customID) {
		return nil, nil, fmt.Errorf("%w: invalid consumer custom_id", ErrBadRequest)
	}
	creds := make([]model.ConsumerCredential, 0, len(in.Credentials))
	for _, c := range in.Credentials {
		plugin, cfg, err := normalizeCredential(c)
		if err != nil {
			return nil, nil, err
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return nil, nil, err
		}
		creds = append(creds, model.ConsumerCredential{Plugin: plugin, Config: datatypes.JSON(raw)})
	}
	return &model.Consumer{SpaceID: spaceID, Username: username, CustomID: customID}, creds, nil
}

func normalizeCredential(in CredentialInput) (string, map[string]string, error) {
	plugin := strings.TrimSpace(strings.ToLower(in.Plugin))
	cfg := map[string]string{}
	for k, v := range in.Config {
		cfg[k] = strings.TrimSpace(v)
	}
	switch plugin {
	case "key-auth":
		if cfg["key"] == "" {
			key, err := randomToken(16)
			if err != nil {
				return "", nil, err
			}
			cfg["key"] = key
		}
	case "basic-auth":
		if cfg["username"] == "" || cfg["password"] == "" {
			return "", nil, fmt.Errorf("%w: basic-auth requires username and password", ErrBadRequest)
		}
	case "jwt":
		algo := strings.ToUpper(cfg["algorithm"])
		if algo == "" {
			algo = "HS256"
		}
		switch algo {
		case "HS256", "HS384", "HS512", "RS256", "RS384", "RS512", "ES256", "ES384":
		default:
			return "", nil, fmt.Errorf("%w: unsupported jwt algorithm", ErrBadRequest)
		}
		cfg["algorithm"] = algo
		if cfg["key"] == "" {
			key, err := randomToken(12)
			if err != nil {
				return "", nil, err
			}
			cfg["key"] = key
		}
		if strings.HasPrefix(algo, "HS") && cfg["secret"] == "" {
			secret, err := randomToken(24)
			if err != nil {
				return "", nil, err
			}
			cfg["secret"] = secret
		}
		if !strings.HasPrefix(algo, "HS") && cfg["rsa_public_key"] == "" {
			return "", nil, fmt.Errorf("%w: jwt %s requires rsa_public_key", ErrBadRequest, algo)
		}
	case "hmac-auth":
		if cfg["username"] == "" {
			return "", nil, fmt.Errorf("%w: hmac-auth requires username", ErrBadRequest)
		}
		if cfg["secret"] == "" {
			secret, err := randomToken(24)
			if err != nil {
				return "", nil, err
			}
			cfg["secret"] = secret
		}
	case "acl":
		if cfg["group"] == "" {
			return "", nil, fmt.Errorf("%w: acl requires group", ErrBadRequest)
		}
	default:
		return "", nil, fmt.Errorf("%w: unsupported credential plugin", ErrBadRequest)
	}
	return plugin, cfg, nil
}

func consumerToSync(c model.Consumer, existingID string) kongclient.ConsumerSync {
	creds := make([]kongclient.CredentialSync, 0, len(c.Credentials))
	for _, cred := range c.Credentials {
		cfg := map[string]string{}
		_ = json.Unmarshal(cred.Config, &cfg)
		creds = append(creds, kongclient.CredentialSync{Plugin: cred.Plugin, Config: cfg})
	}
	customID := ""
	if c.CustomID != "" {
		customID = model.KongConsumerName(c.SpaceID, c.CustomID)
	}
	return kongclient.ConsumerSync{
		KongUsername: model.KongConsumerName(c.SpaceID, c.Username),
		KongCustomID: customID,
		ExistingID:   existingID,
		Tag:          fmt.Sprintf("agm-space-%d", c.SpaceID),
		Credentials:  creds,
	}
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
