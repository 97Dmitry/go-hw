package db

import (
	"net/url"
	"testing"
)

func TestDSN(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://explicit/db")
	if DSN() != "postgres://explicit/db" {
		t.Fatal("DATABASE_URL must take priority")
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_USER", "a@b")
	t.Setenv("DB_PASS", "p:/@ ss")
	t.Setenv("DB_NAME", "cashapp")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	u, err := url.Parse(DSN())
	if err != nil {
		t.Fatal(err)
	}
	pass, _ := u.User.Password()
	if u.User.Username() != "a@b" || pass != "p:/@ ss" || u.Host != "localhost:5432" {
		t.Fatalf("incorrect DSN encoding")
	}
}
