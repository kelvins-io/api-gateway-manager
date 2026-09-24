package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/config"
	"github.com/kelvins-io/api-gateway-manager/internal/database"
	"github.com/kelvins-io/api-gateway-manager/internal/handler"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/jwtutil"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/logger"
	"github.com/kelvins-io/api-gateway-manager/internal/router"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log.Level, cfg.Log.Encoding)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		log.Fatal("connect database", zap.Error(err))
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("auto migrate", zap.Error(err))
	}

	gin.SetMode(cfg.Server.Mode)
	jwtMgr := jwtutil.NewManager(cfg.JWT.Secret, cfg.JWT.ExpireHours)

	authSvc := service.NewAuthService(db, jwtMgr)
	spaceSvc := service.NewSpaceService(db)
	gatewaySvc := service.NewGatewayService(db)
	groupSvc := service.NewGroupService(db)
	apiSvc := service.NewAPIService(db)
	upstreamSvc := service.NewUpstreamService(db)
	consumerSvc := service.NewConsumerService(db)
	pluginSvc := service.NewPluginService(db)

	r := router.Setup(db, jwtMgr, log, router.Handlers{
		Auth:     handler.NewAuthHandler(authSvc),
		Space:    handler.NewSpaceHandler(spaceSvc),
		Gateway:  handler.NewGatewayHandler(gatewaySvc),
		Group:    handler.NewGroupHandler(groupSvc, consumerSvc),
		API:      handler.NewAPIHandler(apiSvc),
		Upstream: handler.NewUpstreamHandler(upstreamSvc),
		Consumer: handler.NewConsumerHandler(consumerSvc),
		Plugin:   handler.NewPluginHandler(pluginSvc),
		APISvc:   apiSvc,
		GroupSvc: groupSvc,
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Info("server starting", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}
