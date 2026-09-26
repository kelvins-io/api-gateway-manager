package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gopkg.in/yaml.v3"
)

var (
	httpMethods = []string{"get", "post", "put", "delete", "patch", "head", "options", "trace"}
	nameSanitize = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
)

type ImportOpenAPIOptions struct {
	Content         []byte
	Filename        string
	DryRun          bool
	ServiceProtocol string
	ServiceHost     string
	ServicePort     int
	ServicePath     string
}

type ImportOpenAPIItem struct {
	Name               string `json:"name"`
	AccessPath         string `json:"access_path"`
	AccessPathPrefixed string `json:"access_path_prefixed"`
	AccessMethods      string `json:"access_methods"`
	AccessProtocols    string `json:"access_protocols"`
	ServiceProtocol    string `json:"service_protocol"`
	ServiceHost        string `json:"service_host"`
	ServicePort        int    `json:"service_port"`
	ServicePath        string `json:"service_path"`
	Action             string `json:"action"` // create | update
	ExistingID         uint64 `json:"existing_id,omitempty"`
}

type ImportOpenAPIFail struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

type ImportOpenAPIResult struct {
	Items   []ImportOpenAPIItem `json:"items"`
	Created []model.API         `json:"created,omitempty"`
	Updated []model.API         `json:"updated,omitempty"`
	Failed  []ImportOpenAPIFail `json:"failed,omitempty"`
	Total   int                 `json:"total"`
}

type openAPIDoc struct {
	OpenAPI string                 `json:"openapi" yaml:"openapi"`
	Swagger string                 `json:"swagger" yaml:"swagger"`
	Info    openAPIInfo            `json:"info" yaml:"info"`
	Servers []openAPIServer        `json:"servers" yaml:"servers"`
	Host    string                 `json:"host" yaml:"host"`
	BasePath string                `json:"basePath" yaml:"basePath"`
	Schemes []string               `json:"schemes" yaml:"schemes"`
	Paths   map[string]json.RawMessage `json:"paths" yaml:"paths"`
}

type openAPIInfo struct {
	Title string `json:"title" yaml:"title"`
}

type openAPIServer struct {
	URL string `json:"url" yaml:"url"`
}

type openAPIOperation struct {
	OperationID string `json:"operationId" yaml:"operationId"`
	Summary     string `json:"summary" yaml:"summary"`
	Servers     []openAPIServer `json:"servers" yaml:"servers"`
}

func (s *APIService) ImportOpenAPI(ctx context.Context, groupID uint64, opt ImportOpenAPIOptions) (*ImportOpenAPIResult, error) {
	group, err := s.getGroup(groupID)
	if err != nil {
		return nil, err
	}
	var space model.Space
	if err := s.db.Select("id", "prefix").First(&space, group.SpaceID).Error; err != nil {
		return nil, err
	}

	items, err := parseOpenAPIDocument(opt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: no API paths found in OpenAPI document", ErrBadRequest)
	}

	existingByName, err := s.apisByNameInGroup(groupID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		prefixed := model.ApplyPathPrefix(space.Prefix, []string{items[i].AccessPath})
		if len(prefixed) > 0 {
			items[i].AccessPathPrefixed = prefixed[0]
		} else {
			items[i].AccessPathPrefixed = items[i].AccessPath
		}
		if existing, ok := existingByName[items[i].Name]; ok {
			items[i].Action = "update"
			items[i].ExistingID = existing.ID
		} else {
			items[i].Action = "create"
		}
	}

	result := &ImportOpenAPIResult{Items: items, Total: len(items)}
	if opt.DryRun {
		return result, nil
	}

	created := make([]model.API, 0)
	updated := make([]model.API, 0)
	failed := make([]ImportOpenAPIFail, 0)
	for _, item := range items {
		if item.Action == "update" {
			existing := existingByName[item.Name]
			port := item.ServicePort
			api, err := s.Update(ctx, existing.ID, UpdateAPIInput{
				AccessPath:      item.AccessPath,
				AccessMethods:   item.AccessMethods,
				AccessProtocols: item.AccessProtocols,
				AccessHosts:     existing.AccessHosts,
				AccessHeaders:   decodeHeaders(existing.AccessHeaders),
				ServiceProtocol: item.ServiceProtocol,
				ServiceHostKind: model.HostKindDirect,
				ServiceHost:     item.ServiceHost,
				ServicePort:     &port,
				ServicePath:     item.ServicePath,
			})
			if err != nil {
				failed = append(failed, ImportOpenAPIFail{
					Name:  item.Name,
					Path:  item.AccessPath,
					Error: err.Error(),
				})
				continue
			}
			updated = append(updated, *api)
			continue
		}
		api, err := s.Create(groupID, CreateAPIInput{
			Name:            item.Name,
			AccessPath:      item.AccessPath,
			AccessMethods:   item.AccessMethods,
			AccessProtocols: item.AccessProtocols,
			ServiceProtocol: item.ServiceProtocol,
			ServiceHostKind: model.HostKindDirect,
			ServiceHost:     item.ServiceHost,
			ServicePort:     item.ServicePort,
			ServicePath:     item.ServicePath,
		})
		if err != nil {
			failed = append(failed, ImportOpenAPIFail{
				Name:  item.Name,
				Path:  item.AccessPath,
				Error: err.Error(),
			})
			continue
		}
		created = append(created, *api)
	}
	result.Created = created
	result.Updated = updated
	result.Failed = failed
	return result, nil
}

func (s *APIService) apisByNameInGroup(groupID uint64) (map[string]model.API, error) {
	var list []model.API
	if err := s.db.Select("id", "name", "access_hosts", "access_headers").
		Where("group_id = ?", groupID).Find(&list).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.API, len(list))
	for _, api := range list {
		out[api.Name] = api
	}
	return out, nil
}

func parseOpenAPIDocument(opt ImportOpenAPIOptions) ([]ImportOpenAPIItem, error) {
	raw, err := decodeOpenAPIBytes(opt.Content)
	if err != nil {
		return nil, err
	}

	var doc openAPIDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI document: %w", err)
	}
	if doc.OpenAPI == "" && doc.Swagger == "" {
		return nil, fmt.Errorf("missing openapi/swagger version field")
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("paths is empty")
	}

	defaultSvc, err := resolveDefaultService(doc, opt)
	if err != nil {
		return nil, err
	}

	pathKeys := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	usedNames := map[string]struct{}{}
	items := make([]ImportOpenAPIItem, 0, len(pathKeys))
	for _, pathKey := range pathKeys {
		ops, pathServers, err := decodePathItem(doc.Paths[pathKey])
		if err != nil {
			return nil, fmt.Errorf("path %s: %w", pathKey, err)
		}
		if len(ops) == 0 {
			continue
		}
		methods := make([]string, 0, len(ops))
		for _, m := range httpMethods {
			if _, ok := ops[m]; ok {
				methods = append(methods, strings.ToUpper(m))
			}
		}
		if len(methods) == 0 {
			continue
		}

		accessPath := normalizeOpenAPIPath(pathKey)
		if accessPath == "" {
			continue
		}

		name := pickOperationName(doc.Info.Title, accessPath, methods, ops)
		name = uniqueImportName(name, usedNames)

		svc := defaultSvc
		if len(pathServers) > 0 && strings.TrimSpace(pathServers[0].URL) != "" {
			if parsed, err := parseServerURL(pathServers[0].URL); err == nil {
				svc = mergeServiceOverride(parsed, opt)
			}
		}
		// Prefer operation-level servers from the first method that defines them.
		for _, m := range httpMethods {
			op, ok := ops[m]
			if !ok || len(op.Servers) == 0 {
				continue
			}
			if parsed, err := parseServerURL(op.Servers[0].URL); err == nil {
				svc = mergeServiceOverride(parsed, opt)
			}
			break
		}

		items = append(items, ImportOpenAPIItem{
			Name:            name,
			AccessPath:      accessPath,
			AccessMethods:   strings.Join(methods, ","),
			AccessProtocols: svc.Protocol,
			ServiceProtocol: svc.Protocol,
			ServiceHost:     svc.Host,
			ServicePort:     svc.Port,
			ServicePath:     svc.ServicePath,
		})
	}
	return items, nil
}

type resolvedService struct {
	Protocol    string
	Host        string
	Port        int
	ServicePath string
}

func resolveDefaultService(doc openAPIDoc, opt ImportOpenAPIOptions) (resolvedService, error) {
	base := resolvedService{
		Protocol:    "http",
		Host:        "httpbin.org",
		Port:        80,
		ServicePath: "/",
	}
	if doc.OpenAPI != "" && len(doc.Servers) > 0 && strings.TrimSpace(doc.Servers[0].URL) != "" {
		if parsed, err := parseServerURL(doc.Servers[0].URL); err == nil {
			base = parsed
		}
	} else if doc.Swagger != "" {
		if host := strings.TrimSpace(doc.Host); host != "" {
			base.Host = host
			if h, p, ok := splitHostPort(host); ok {
				base.Host = h
				base.Port = p
			}
		}
		if len(doc.Schemes) > 0 {
			base.Protocol = strings.ToLower(strings.TrimSpace(doc.Schemes[0]))
		}
		if bp := strings.TrimSpace(doc.BasePath); bp != "" {
			if !strings.HasPrefix(bp, "/") {
				bp = "/" + bp
			}
			base.ServicePath = bp
		}
		if base.Port == 80 && base.Protocol == "https" {
			base.Port = 443
		}
	}
	return mergeServiceOverride(base, opt), nil
}

func mergeServiceOverride(base resolvedService, opt ImportOpenAPIOptions) resolvedService {
	out := base
	if p := strings.TrimSpace(opt.ServiceProtocol); p != "" {
		out.Protocol = strings.ToLower(p)
	}
	if h := strings.TrimSpace(opt.ServiceHost); h != "" {
		out.Host = h
	}
	if opt.ServicePort > 0 {
		out.Port = opt.ServicePort
	}
	if p := strings.TrimSpace(opt.ServicePath); p != "" {
		out.ServicePath = p
	}
	if out.Port == 0 {
		if out.Protocol == "https" || out.Protocol == "grpcs" {
			out.Port = 443
		} else {
			out.Port = 80
		}
	}
	if out.ServicePath == "" {
		out.ServicePath = "/"
	}
	return out
}

func parseServerURL(raw string) (resolvedService, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return resolvedService{}, fmt.Errorf("empty server url")
	}
	// OpenAPI may use relative server URLs; treat as path only.
	if strings.HasPrefix(raw, "/") {
		return resolvedService{Protocol: "http", Host: "httpbin.org", Port: 80, ServicePath: raw}, nil
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return resolvedService{}, err
	}
	out := resolvedService{
		Protocol:    strings.ToLower(u.Scheme),
		Host:        u.Hostname(),
		ServicePath: u.EscapedPath(),
	}
	if out.Protocol == "" {
		out.Protocol = "http"
	}
	if out.Host == "" {
		out.Host = "httpbin.org"
	}
	if out.ServicePath == "" {
		out.ServicePath = "/"
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return resolvedService{}, err
		}
		out.Port = port
	} else if out.Protocol == "https" || out.Protocol == "grpcs" {
		out.Port = 443
	} else {
		out.Port = 80
	}
	return out, nil
}

func splitHostPort(host string) (string, int, bool) {
	if !strings.Contains(host, ":") {
		return host, 0, false
	}
	h, p, err := netSplitHostPort(host)
	if err != nil {
		return host, 0, false
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return host, 0, false
	}
	return h, port, true
}

func netSplitHostPort(hostport string) (host, port string, err error) {
	// Avoid importing net just for SplitHostPort edge cases with IPv6; simple last-colon split for host:port.
	i := strings.LastIndex(hostport, ":")
	if i < 0 {
		return "", "", fmt.Errorf("missing port")
	}
	return hostport[:i], hostport[i+1:], nil
}

func decodeOpenAPIBytes(content []byte) ([]byte, error) {
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	if content[0] == '{' || content[0] == '[' {
		return content, nil
	}
	var node interface{}
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("invalid YAML/JSON: %w", err)
	}
	raw, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func decodePathItem(raw json.RawMessage) (map[string]openAPIOperation, []openAPIServer, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil, nil
	}
	// Path Item must be an object; skip $ref-only / malformed arrays without failing whole import.
	if trimmed[0] != '{' {
		return nil, nil, nil
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, nil, err
	}
	var pathServers []openAPIServer
	if body, ok := generic["servers"]; ok && len(body) > 0 && string(body) != "null" {
		_ = json.Unmarshal(body, &pathServers)
	}
	out := map[string]openAPIOperation{}
	for _, m := range httpMethods {
		body, ok := generic[m]
		if !ok {
			continue
		}
		op, ok := decodeOperation(body)
		if !ok {
			continue
		}
		out[m] = op
	}
	return out, pathServers, nil
}

// decodeOperation accepts a standard Operation Object. Some non-standard docs wrap
// the operation in a single-element array; those are unwrapped. Other non-object
// values are ignored so one bad method doesn't abort the whole import.
func decodeOperation(raw json.RawMessage) (openAPIOperation, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return openAPIOperation{}, false
	}
	switch trimmed[0] {
	case '{':
		var op openAPIOperation
		if err := json.Unmarshal(trimmed, &op); err != nil {
			return openAPIOperation{}, false
		}
		return op, true
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return openAPIOperation{}, false
		}
		for _, item := range arr {
			if op, ok := decodeOperation(item); ok {
				return op, true
			}
		}
		return openAPIOperation{}, false
	default:
		return openAPIOperation{}, false
	}
}

func normalizeOpenAPIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Keep {param} templates as-is for user editing after import.
	return path
}

func pickOperationName(title, path string, methods []string, ops map[string]openAPIOperation) string {
	if len(methods) == 1 {
		op := ops[strings.ToLower(methods[0])]
		if id := strings.TrimSpace(op.OperationID); id != "" {
			return truncateName(id)
		}
		if sum := strings.TrimSpace(op.Summary); sum != "" {
			return truncateName(sum)
		}
	}
	base := strings.Trim(path, "/")
	if base == "" {
		base = "root"
	}
	base = nameSanitize.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "api"
	}
	if title = strings.TrimSpace(title); title != "" {
		title = nameSanitize.ReplaceAllString(title, "-")
		title = strings.Trim(title, "-")
		if title != "" {
			base = title + "-" + base
		}
	}
	return truncateName(base)
}

func uniqueImportName(name string, used map[string]struct{}) string {
	name = truncateName(strings.TrimSpace(name))
	if name == "" {
		name = "imported-api"
	}
	candidate := name
	for i := 2; ; i++ {
		if _, ok := used[strings.ToLower(candidate)]; !ok {
			used[strings.ToLower(candidate)] = struct{}{}
			return candidate
		}
		suffix := "-" + strconv.Itoa(i)
		candidate = truncateName(name, len(suffix)) + suffix
	}
}

func truncateName(name string, reserve ...int) string {
	name = strings.TrimSpace(name)
	max := 128
	if len(reserve) > 0 {
		max -= reserve[0]
		if max < 1 {
			max = 1
		}
	}
	runes := []rune(name)
	if len(runes) > max {
		return string(runes[:max])
	}
	return name
}
