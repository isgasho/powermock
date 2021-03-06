package internal

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/storyicon/powermock/pkg/apimanager"
	grpcmockserver "github.com/storyicon/powermock/pkg/mockserver/grpc"
	httpmockserver "github.com/storyicon/powermock/pkg/mockserver/http"
	"github.com/storyicon/powermock/pkg/pluginregistry"
	pluginsgrpc "github.com/storyicon/powermock/pkg/pluginregistry/grpc"
	pluginshttp "github.com/storyicon/powermock/pkg/pluginregistry/http"
	pluginscript "github.com/storyicon/powermock/pkg/pluginregistry/script"
	pluginssimple "github.com/storyicon/powermock/pkg/pluginregistry/simple"
	pluginredis "github.com/storyicon/powermock/pkg/pluginregistry/storage/redis"
	"github.com/storyicon/powermock/pkg/util/logger"
)

// Startup is used to start up application
func Startup(
	ctx context.Context, cancelFunc context.CancelFunc,
	cfg *Config, log logger.Logger, registerer prometheus.Registerer) error {

	log.LogInfo(nil, "* start to create pluginRegistry")
	pluginRegistry, err := pluginregistry.New(cfg.PluginRegistry, log, registerer)
	if err != nil {
		return err
	}

	log.LogInfo(nil, "* start to create apiManager")
	apiManager, err := apimanager.New(cfg.ApiManager,
		pluginRegistry,
		log, registerer)
	if err != nil {
		log.LogFatal(nil, "failed to create apiManager: %s", err)
	}

	var (
		mockPlugins    []pluginregistry.MockPlugin
		matchPlugins   []pluginregistry.MatchPlugin
		storagePlugins []pluginregistry.StoragePlugin
		httpMockServer httpmockserver.Provider
		gRPCMockServer grpcmockserver.Provider
	)

	if cfg.HTTPMockServer.IsEnabled() {
		log.LogInfo(nil, "* start to create httpMockServer")
		server, err := httpmockserver.New(
			cfg.HTTPMockServer,
			apiManager,
			log,
			prometheus.DefaultRegisterer,
		)
		if err != nil {
			log.LogFatal(nil, "failed to create httpMockServer: %s", err)
		}
		httpMockServer = server
	}

	if cfg.Plugin.Redis.IsEnabled() {
		log.LogInfo(nil, "* start to create plugin(redis)")
		storagePlugin, err := pluginredis.New(cfg.Plugin.Redis, log, registerer)
		if err != nil {
			return err
		}
		storagePlugins = append(storagePlugins, storagePlugin)
	}

	if cfg.Plugin.Simple.IsEnabled() {
		log.LogInfo(nil, "* start to create plugin(simple)")
		simplePlugin, err := pluginssimple.New(cfg.Plugin.Simple, log, registerer)
		if err != nil {
			return err
		}
		mockPlugins = append(mockPlugins, simplePlugin)
		matchPlugins = append(matchPlugins, simplePlugin)
	}

	if cfg.Plugin.Script.IsEnabled() {
		log.LogInfo(nil, "* start to create plugin(script)")
		scriptPlugin, err := pluginscript.New(cfg.Plugin.Script, log, registerer)
		if err != nil {
			return err
		}
		mockPlugins = append(mockPlugins, scriptPlugin)
		matchPlugins = append(matchPlugins, scriptPlugin)
	}

	if cfg.GRPCMockServer.IsEnabled() {
		log.LogInfo(nil, "* start to create grpcMockServer")
		server, err := grpcmockserver.New(
			cfg.GRPCMockServer,
			apiManager,
			log,
			prometheus.DefaultRegisterer,
		)
		if err != nil {
			log.LogFatal(nil, "failed to create gRPCMockServer:", err)
		}
		gRPCMockServer = server
		if cfg.Plugin.GRPC.IsEnabled() {
			log.LogInfo(nil, "* start to create plugin(gRPC)")
			grpcPlugin, err := pluginsgrpc.New(cfg.Plugin.GRPC, server.GetProtoManager().GetMethod, log, registerer)
			if err != nil {
				return err
			}
			mockPlugins = append(mockPlugins, grpcPlugin)
		}
	}

	if cfg.Plugin.HTTP.IsEnabled() {
		log.LogInfo(nil, "* start to create plugin(http)")
		httpPlugin, err := pluginshttp.New(cfg.Plugin.HTTP, log, registerer)
		if err != nil {
			return err
		}
		mockPlugins = append(mockPlugins, httpPlugin)
	}

	log.LogInfo(nil, "* start to install plugins")
	if err := pluginRegistry.RegisterMockPlugins(mockPlugins...); err != nil {
		return err
	}
	if err := pluginRegistry.RegisterMatchPlugins(matchPlugins...); err != nil {
		return err
	}
	if err := pluginRegistry.RegisterStoragePlugins(storagePlugins...); err != nil {
		return err
	}

	log.LogInfo(nil, "* start to start apiManager")
	if err := apiManager.Start(ctx, cancelFunc); err != nil {
		return err
	}

	if cfg.GRPCMockServer.IsEnabled() && gRPCMockServer != nil {
		log.LogInfo(nil, "* start to start gRPCMockServer")
		if err := gRPCMockServer.Start(ctx, cancelFunc); err != nil {
			return err
		}
	}

	if cfg.HTTPMockServer.IsEnabled() && httpMockServer != nil {
		log.LogInfo(nil, "* start to start httpMockServer")
		if err := httpMockServer.Start(ctx, cancelFunc); err != nil {
			return err
		}
	}
	return nil
}
