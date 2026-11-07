package conf

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	validDev := func() *Bootstrap {
		return &Bootstrap{
			Server: &Server{Env: "dev"},
			Data: &Data{Database: &Data_Database{
				Driver: "mysql",
				Source: "root:root@tcp(127.0.0.1:3306)/planstar_admin?parseTime=True",
			}},
			Auth: &Auth{JwtSecret: "dev-secret"},
		}
	}
	validProd := func() *Bootstrap {
		return &Bootstrap{
			Server: &Server{Env: "prod"},
			Data: &Data{Database: &Data_Database{
				Driver:      "mysql",
				Source:      "planstar_admin:secret@tcp(mysql:3306)/planstar_admin?parseTime=True",
				AutoMigrate: false,
			}},
			Auth:     &Auth{JwtSecret: strings.Repeat("s", 32), CookieSecure: true},
			Planstar: &Planstar{OpsToken: "ops-token"},
		}
	}

	tests := []struct {
		name    string
		conf    func() *Bootstrap
		wantErr string
	}{
		{name: "dev valid", conf: validDev},
		{name: "prod valid", conf: validProd},
		{
			name: "missing database driver",
			conf: func() *Bootstrap {
				c := validDev()
				c.Data.Database.Driver = ""
				return c
			},
			wantErr: "data.database.driver",
		},
		{
			name: "unsupported database driver",
			conf: func() *Bootstrap {
				c := validDev()
				c.Data.Database.Driver = "postgres"
				return c
			},
			wantErr: "data.database.driver",
		},
		{
			name: "missing database source",
			conf: func() *Bootstrap {
				c := validDev()
				c.Data.Database.Source = ""
				return c
			},
			wantErr: "data.database.source",
		},
		{
			name: "missing jwt secret",
			conf: func() *Bootstrap {
				c := validDev()
				c.Auth.JwtSecret = ""
				return c
			},
			wantErr: "auth.jwt_secret",
		},
		{
			name: "prod short jwt secret",
			conf: func() *Bootstrap {
				c := validProd()
				c.Auth.JwtSecret = "too-short"
				return c
			},
			wantErr: "at least 32",
		},
		{
			name: "prod insecure cookie",
			conf: func() *Bootstrap {
				c := validProd()
				c.Auth.CookieSecure = false
				return c
			},
			wantErr: "auth.cookie_secure",
		},
		{
			name: "prod auto migrate",
			conf: func() *Bootstrap {
				c := validProd()
				c.Data.Database.AutoMigrate = true
				return c
			},
			wantErr: "auto_migrate",
		},
		{
			name: "prod missing ops token",
			conf: func() *Bootstrap {
				c := validProd()
				c.Planstar.OpsToken = ""
				return c
			},
			wantErr: "planstar.ops_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.conf())
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNilBootstrap(t *testing.T) {
	if err := Validate(nil); err == nil {
		t.Fatal("Validate(nil) = nil, want error")
	}
}
