package model

import "testing"

func TestNormalizePrefix(t *testing.T) {
	got, err := NormalizePrefix("order")
	if err != nil || got != "/order" {
		t.Fatalf("got %q err %v", got, err)
	}
	got, err = NormalizePrefix("/team/order/")
	if err != nil || got != "/team/order" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := NormalizePrefix("/"); err == nil {
		t.Fatal("expected error for /")
	}
}

func TestApplyPathPrefix(t *testing.T) {
	got := ApplyPathPrefix("/order", SplitPaths("/users,/order/items"))
	if len(got) != 2 || got[0] != "/order/users" || got[1] != "/order/items" {
		t.Fatalf("unexpected %v", got)
	}
	got = ApplyPathPrefix("", SplitPaths("/users"))
	if len(got) != 1 || got[0] != "/users" {
		t.Fatalf("unexpected %v", got)
	}
}

func TestApplyPathPrefixMovesTildeToFront(t *testing.T) {
	got := ApplyPathPrefix("/order", SplitPaths("~/files/*,~items*"))
	if len(got) != 2 || got[0] != "~/order/files/*" || got[1] != "~/order/items*" {
		t.Fatalf("unexpected %v", got)
	}
	// Clients that force '/' before '~' should still work.
	got = ApplyPathPrefix("/order", SplitPaths("/~/files/*"))
	if len(got) != 1 || got[0] != "~/order/files/*" {
		t.Fatalf("unexpected %v", got)
	}
	// Already-prefixed full path keeps '~' at front and does not double prefix.
	got = ApplyPathPrefix("/order", SplitPaths("~/order/files/*"))
	if len(got) != 1 || got[0] != "~/order/files/*" {
		t.Fatalf("unexpected %v", got)
	}
}

func TestSplitPathsPreservesTilde(t *testing.T) {
	got := SplitPaths("~/files/*,/users")
	if len(got) != 2 || got[0] != "~/files/*" || got[1] != "/users" {
		t.Fatalf("unexpected %v", got)
	}
}

func TestKongRoutePaths(t *testing.T) {
	got := KongRoutePaths([]string{"/users", "/files/*", "/order/items*"})
	if len(got) != 3 || got[0] != "/users" || got[1] != "~/files/*" || got[2] != "~/order/items*" {
		t.Fatalf("unexpected %v", got)
	}
	got = KongRoutePaths([]string{"~/already/*"})
	if len(got) != 1 || got[0] != "~/already/*" {
		t.Fatalf("should not double-prefix: %v", got)
	}
}
