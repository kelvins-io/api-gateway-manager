package service

import (
	"testing"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
)

func TestBuildConsumer(t *testing.T) {
	c, creds, err := buildConsumer(3, UpsertConsumerInput{
		Username: "app-a",
		CustomID: "tenant.1",
		Credentials: []CredentialInput{
			{Plugin: "key-auth", Config: map[string]string{}},
			{Plugin: "acl", Config: map[string]string{"group": "gold"}},
			{Plugin: "jwt", Config: map[string]string{"algorithm": "hs256"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Username != "app-a" || c.SpaceID != 3 || c.CustomID != "tenant.1" {
		t.Fatalf("consumer = %+v", c)
	}
	if len(creds) != 3 {
		t.Fatalf("creds = %d", len(creds))
	}
	sync := consumerToSync(*c, "")
	if sync.KongUsername != "agm-s3-app-a" || sync.KongCustomID != "agm-s3-tenant.1" {
		t.Fatalf("sync name = %s %s", sync.KongUsername, sync.KongCustomID)
	}
	if _, _, err := buildConsumer(1, UpsertConsumerInput{Username: "bad name"}); err == nil {
		t.Fatal("expected invalid username")
	}
	if _, _, err := buildConsumer(1, UpsertConsumerInput{
		Username:    "ok",
		Credentials: []CredentialInput{{Plugin: "basic-auth", Config: map[string]string{"username": "u"}}},
	}); err == nil {
		t.Fatal("expected basic-auth error")
	}
}

func TestValidateCredentialUpdate(t *testing.T) {
	current := []model.ConsumerCredential{
		{Plugin: "key-auth"},
		{Plugin: "acl", Config: []byte(`{"group":"G-9"}`)},
		{Plugin: "acl", Config: []byte(`{"group":"gold"}`)},
	}
	same := []model.ConsumerCredential{{Plugin: "key-auth"}, {Plugin: "acl", Config: []byte(`{"group":"gold"}`)}}
	if err := validateCredentialUpdate(current, same); err != nil {
		t.Fatal(err)
	}
	if err := validateCredentialUpdate(current, []model.ConsumerCredential{{Plugin: "jwt"}, {Plugin: "acl"}}); err == nil {
		t.Fatal("expected reject type change")
	}
	if err := validateCredentialUpdate(current, []model.ConsumerCredential{{Plugin: "key-auth"}}); err == nil {
		t.Fatal("expected reject removal")
	}
	if err := validateCredentialUpdate(current, append(same, model.ConsumerCredential{Plugin: "basic-auth"})); err == nil {
		t.Fatal("expected reject addition")
	}
}

func TestWithAPIAuth(t *testing.T) {
	api := &model.API{
		ID:          4,
		AuthEnabled: true,
		AuthPlugin:  "key-auth",
		AuthConfig:  []byte(`{"key_names":["apikey"],"hide_credentials":false,"key_in_header":true,"key_in_query":true,"key_in_body":false}`),
	}
	specs, err := withAPIAuth(api, []kongclient.PluginSync{{Name: "cors", Config: map[string]interface{}{}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 3 || specs[0].Name != "cors" || specs[1].Name != "key-auth" || specs[2].Name != "acl" {
		t.Fatalf("specs = %+v", specs)
	}
	allow, _ := specs[2].Config["allow"].([]string)
	if len(allow) != 1 || allow[0] != "G-4" {
		t.Fatalf("allow = %#v", specs[2].Config["allow"])
	}
	if err := validateAuthUpdate(false, "", true, "key-auth"); err == nil {
		t.Fatal("expected reject enabling auth")
	}
	if err := validateAuthUpdate(true, "key-auth", true, "jwt"); err == nil {
		t.Fatal("expected reject auth type change")
	}
	if err := validateAuthUpdate(true, "key-auth", true, "key-auth"); err != nil {
		t.Fatal(err)
	}
	plain, err := withAPIAuth(&model.API{ID: 1}, specs)
	if err != nil || len(plain) != 3 {
		t.Fatalf("disabled auth changed specs: %v %+v", err, plain)
	}
}
