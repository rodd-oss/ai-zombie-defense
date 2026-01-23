package handlers_test

import (
	"ai-zombie-defense/server/internal/api/gateway"
	"ai-zombie-defense/server/internal/testutils"
	"database/sql"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	"testing"
)

func createFullTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := testutils.GetTestConfig()
	gw := gateway.NewAPIGateway(cfg, logger, db)
	return gw.Router()
}
