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
