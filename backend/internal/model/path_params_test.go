package model

import "testing"

func TestExtractPathParams(t *testing.T) {
	got := ExtractPathParams("/a/{id}/b/{name}/c/{id}")
	if len(got) != 2 || got[0] != "id" || got[1] != "name" {
		t.Fatalf("got %v", got)
	}
}

func TestToURICaptureTemplate(t *testing.T) {
	got := ToURICaptureTemplate("/api/v1/spaces/{id}/groups/{gid}")
	want := `/api/v1/spaces/$(uri_captures["id"])/groups/$(uri_captures["gid"])`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateAccessAndServicePathParams(t *testing.T) {
	if err := ValidateAccessAndServicePathParams(
		[]string{"/spaces/{id}/groups"},
		"/api/v1/spaces/{id}/groups",
	); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := ValidateAccessAndServicePathParams(
		[]string{"/spaces/{id}/groups"},
		"/api/v1/spaces/groups",
	); err == nil {
		t.Fatal("expected mismatch error")
	}
	if err := ValidateAccessAndServicePathParams(
		[]string{"/spaces/{id}", "/health"},
		"/spaces/{id}",
	); err == nil {
		t.Fatal("expected mixed-path error")
	}
	if err := ValidateAccessAndServicePathParams(
		[]string{"/users"},
		"/users/{id}",
	); err == nil {
		t.Fatal("expected service-only param error")
	}
}

func TestJoinServiceAccessPath(t *testing.T) {
	if got := JoinServiceAccessPath("/api/v1", "/spaces/{id}"); got != "/api/v1/spaces/{id}" {
		t.Fatalf("got %q", got)
	}
	if got := JoinServiceAccessPath("/", "/spaces/{id}"); got != "/spaces/{id}" {
		t.Fatalf("got %q", got)
	}
}
