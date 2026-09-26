package service

import (
	"strings"
	"testing"
)

func TestParseOpenAPIJSON(t *testing.T) {
	doc := []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "Petstore"},
  "servers": [{"url": "https://petstore.example.com/v1"}],
  "paths": {
    "/pets": {
      "get": {"operationId": "listPets", "summary": "List pets"},
      "post": {"operationId": "createPet"}
    },
    "/pets/{petId}": {
      "get": {"summary": "Get a pet"}
    }
  }
}`)
	items, err := parseOpenAPIDocument(ImportOpenAPIOptions{Content: doc})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].AccessPath != "/pets" || items[0].AccessMethods != "GET,POST" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if items[0].Name != "Petstore-pets" {
		t.Fatalf("want name Petstore-pets, got %s", items[0].Name)
	}
	if items[1].Name != "Get a pet" {
		t.Fatalf("want name from summary, got %s", items[1].Name)
	}
	if items[0].ServiceHost != "petstore.example.com" || items[0].ServicePort != 443 || items[0].ServiceProtocol != "https" {
		t.Fatalf("unexpected service: %+v", items[0])
	}
	if items[0].ServicePath != "/v1" {
		t.Fatalf("want service path /v1, got %s", items[0].ServicePath)
	}
	if items[1].AccessPath != "/pets/{petId}" || items[1].AccessMethods != "GET" {
		t.Fatalf("unexpected second item: %+v", items[1])
	}
}

func TestParseOpenAPIYAMLSwagger2(t *testing.T) {
	doc := []byte(`
swagger: "2.0"
info:
  title: Demo
host: api.example.com:8080
basePath: /api
schemes:
  - http
paths:
  /users:
    get:
      operationId: listUsers
    post:
      summary: Create user
`)
	items, err := parseOpenAPIDocument(ImportOpenAPIOptions{Content: doc})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].AccessPath != "/users" || !strings.Contains(items[0].AccessMethods, "GET") {
		t.Fatalf("unexpected item: %+v", items[0])
	}
	if items[0].ServiceHost != "api.example.com" || items[0].ServicePort != 8080 {
		t.Fatalf("unexpected host/port: %+v", items[0])
	}
	if items[0].ServicePath != "/api" {
		t.Fatalf("want /api, got %s", items[0].ServicePath)
	}
}

func TestParseOpenAPIOperationArrayWrapped(t *testing.T) {
	doc := []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "Cache"},
  "paths": {
    "/cache/{key}": {
      "parameters": [{"name": "key", "in": "path"}],
      "get": [{"operationId": "getCache", "summary": "Get cache"}],
      "delete": {"operationId": "deleteCache"}
    }
  }
}`)
	items, err := parseOpenAPIDocument(ImportOpenAPIOptions{Content: doc})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].AccessPath != "/cache/{key}" {
		t.Fatalf("unexpected path %s", items[0].AccessPath)
	}
	if items[0].AccessMethods != "GET,DELETE" {
		t.Fatalf("unexpected methods %s", items[0].AccessMethods)
	}
}

func TestParseOpenAPISkipsNonObjectPathItem(t *testing.T) {
	doc := []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "X"},
  "paths": {
    "/bad": [],
    "/ok": {"get": {"operationId": "ok"}}
  }
}`)
	items, err := parseOpenAPIDocument(ImportOpenAPIOptions{Content: doc})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].AccessPath != "/ok" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestParseOpenAPIServiceOverride(t *testing.T) {
	doc := []byte(`{
  "openapi": "3.0.0",
  "info": {"title": "X"},
  "paths": {
    "/a": {"get": {"operationId": "getA"}}
  }
}`)
	items, err := parseOpenAPIDocument(ImportOpenAPIOptions{
		Content:         doc,
		ServiceHost:     "backend.local",
		ServicePort:     9000,
		ServiceProtocol: "http",
		ServicePath:     "/svc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].ServiceHost != "backend.local" || items[0].ServicePort != 9000 || items[0].ServicePath != "/svc" {
		t.Fatalf("override ignored: %+v", items[0])
	}
}
