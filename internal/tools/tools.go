//go:build tools

// Package tools keeps infrastructure dependencies pinned in go.mod while the
// mall adapters are being built out. `go mod tidy` considers all build tags,
// so these blank imports survive until the real implementations land.
package tools

import (
	_ "github.com/mojocn/base64Captcha"
	_ "github.com/olivere/elastic/v7"
	_ "github.com/rabbitmq/amqp091-go"
	_ "github.com/redis/go-redis/v9"
	_ "golang.org/x/crypto/pbkdf2"
)
