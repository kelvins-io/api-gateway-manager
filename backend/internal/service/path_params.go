package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func pathParamPluginName(apiID uint64) string {
	return fmt.Sprintf("agm-uri-%d", apiID)
}

func IsPathParamPluginName(name string) bool {
	return strings.HasPrefix(name, "agm-uri-")
}

func validateAPIPathParams(accessPath, servicePath string) error {
	paths := model.SplitPaths(accessPath)
	if err := model.ValidateAccessAndServicePathParams(paths, servicePath); err != nil {
		return fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	return nil
}

// syncPathParamPlugin creates/updates/removes the managed request-transformer that
// rewrites upstream URI with $(uri_captures["name"]) for OpenAPI-style {name} paths.
func (s *APIService) syncPathParamPlugin(api *model.API, accessPath, servicePath string) error {
	if api == nil || api.ID == 0 {
		return fmt.Errorf("%w: api id required", ErrBadRequest)
	}
	spaceID := uint64(0)
	if api.Group != nil {
		spaceID = api.Group.SpaceID
	}
	if spaceID == 0 {
		var group model.APIGroup
		if err := s.db.Select("id", "space_id").First(&group, api.GroupID).Error; err != nil {
			return err
		}
		spaceID = group.SpaceID
	}

	name := pathParamPluginName(api.ID)
	needs := false
	for _, p := range model.SplitPaths(accessPath) {
		if model.HasPathParams(p) {
			needs = true
			break
		}
	}

	var existing model.Plugin
	err := s.db.Where("space_id = ? AND name = ?", spaceID, name).First(&existing).Error
	found := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if !needs {
		if !found {
			return nil
		}
		if err := s.db.Model(&model.API{ID: api.ID}).Association("Plugins").Delete(&existing); err != nil {
			return err
		}
		return s.db.Delete(&existing).Error
	}

	cfg := pathParamTransformerConfig(servicePath)
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	if found {
		if err := s.db.Model(&existing).Updates(map[string]interface{}{
			"plugin":  "request-transformer",
			"config":  datatypes.JSON(raw),
			"enabled": true,
		}).Error; err != nil {
			return err
		}
	} else {
		existing = model.Plugin{
			SpaceID: spaceID,
			Name:    name,
			Plugin:  "request-transformer",
			Config:  datatypes.JSON(raw),
			Enabled: true,
		}
		if err := s.db.Select("SpaceID", "Name", "Plugin", "Config", "Enabled").Create(&existing).Error; err != nil {
			return err
		}
	}

	var linked int64
	if err := s.db.Table("api_plugins").
		Where("api_id = ? AND plugin_id = ?", api.ID, existing.ID).
		Count(&linked).Error; err != nil {
		return err
	}
	if linked == 0 {
		if err := s.db.Model(&model.API{ID: api.ID}).Association("Plugins").Append(&existing); err != nil {
			return err
		}
	}
	return nil
}

func pathParamTransformerConfig(servicePath string) map[string]interface{} {
	uri := model.ToURICaptureTemplate(servicePath)
	emptyNames := []string{}
	emptyPairs := []string{}
	return map[string]interface{}{
		"remove": map[string]interface{}{
			"headers":     emptyNames,
			"querystring": emptyNames,
			"body":        emptyNames,
		},
		"rename": map[string]interface{}{
			"headers":     emptyPairs,
			"querystring": emptyPairs,
			"body":        emptyPairs,
		},
		"replace": map[string]interface{}{
			"headers":     emptyPairs,
			"querystring": emptyPairs,
			"body":        emptyPairs,
			"uri":         uri,
		},
		"add": map[string]interface{}{
			"headers":     emptyPairs,
			"querystring": emptyPairs,
			"body":        emptyPairs,
		},
		"append": map[string]interface{}{
			"headers":     emptyPairs,
			"querystring": emptyPairs,
			"body":        emptyPairs,
		},
	}
}
