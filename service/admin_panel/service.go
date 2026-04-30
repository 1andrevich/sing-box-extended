//go:build with_admin_panel

package admin_panel

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/lib/pq"
	"golang.org/x/net/http2"

	_ "github.com/GoAdminGroup/go-admin/adapter/chi"
	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/sqlite"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	_ "github.com/GoAdminGroup/themes/adminlte"
	_ "github.com/GoAdminGroup/themes/sword"

	"github.com/sagernet/sing-box/adapter"
	boxService "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/common/listener"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/service/admin_panel/migration"
	"github.com/sagernet/sing-box/service/admin_panel/pages"
	"github.com/sagernet/sing-box/service/admin_panel/tables"
	CM "github.com/sagernet/sing-box/service/manager/constant"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	N "github.com/sagernet/sing/common/network"
	aTLS "github.com/sagernet/sing/common/tls"
	"github.com/sagernet/sing/service"
)

func RegisterService(registry *boxService.Registry) {
	boxService.Register[option.AdminPanelServiceOptions](registry, C.TypeAdminPanel, NewService)
}

type Service struct {
	boxService.Adapter
	ctx        context.Context
	logger     log.ContextLogger
	listener   *listener.Listener
	tlsConfig  tls.ServerConfig
	httpServer *http.Server
	options    option.AdminPanelServiceOptions
}

func NewService(ctx context.Context, logger log.ContextLogger, tag string, options option.AdminPanelServiceOptions) (adapter.Service, error) {
	s := &Service{
		Adapter: boxService.NewAdapter(C.TypeAdminPanel, tag),
		ctx:     ctx,
		logger:  logger,
		listener: listener.New(listener.Options{
			Context: ctx,
			Logger:  logger,
			Network: []string{N.NetworkTCP},
			Listen:  options.ListenOptions,
		}),
		options: options,
	}
	return s, nil
}

func (s *Service) Start(stage adapter.StartStage) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	boxManager := service.FromContext[adapter.ServiceManager](s.ctx)
	service, ok := boxManager.Get(s.options.Manager)
	if !ok {
		return E.New("manager ", s.options.Manager, " not found")
	}
	manager, ok := service.(CM.Manager)
	if !ok {
		return E.New("invalid ", s.options.Manager, " manager")
	}
	switch s.options.Database.Driver {
	case "postgresql":
		db, err := sql.Open("postgres", s.options.Database.DSN)
		if err != nil {
			return err
		}
		defer db.Close()
		if err := migration.MigratePostgreSQL(db); err != nil && err != migrate.ErrNoChange {
			return err
		}
	default:
		return E.New("unknown driver \"", s.options.Database.Driver, "\"")
	}
	var generators = map[string]table.Generator{
		"squads": tables.SquadTableFactory(
			manager,
			s.logger,
		),
		"nodes": tables.NodeTableFactory(
			manager,
			s.logger,
		),
		"users": tables.UserTableFactory(
			manager,
			s.logger,
		),
		"connection_limiters": tables.ConnectionLimiterTableFactory(
			manager,
			s.logger,
		),
		"bandwidth_limiters": tables.BandwidthLimiterTableFactory(
			manager,
			s.logger,
		),
	}
	eng := engine.Default()
	chiRouter := chi.NewRouter()
	template.AddComp(chartjs.NewChart())
	if err := eng.AddConfig(&config.Config{
		UrlPrefix: "admin",
		IndexUrl:  "/",
		LoginUrl:  "/login",
		Databases: config.DatabaseList{
			"default": config.Database{
				Driver: s.options.Database.Driver,
				Dsn:    s.options.Database.DSN,
			},
		},
	}).
		AddGenerators(generators).
		Use(chiRouter); err != nil {
		return err
	}
	eng.HTML("GET", "/admin", pages.DashboardPage)
	chiRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusMovedPermanently)
	})
	chiRouter.Get("/admin/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusMovedPermanently)
	})
	if s.options.TLS != nil {
		tlsConfig, err := tls.NewServer(s.ctx, s.logger, common.PtrValueOrDefault(s.options.TLS))
		if err != nil {
			return err
		}
		s.tlsConfig = tlsConfig
	}
	if s.tlsConfig != nil {
		err := s.tlsConfig.Start()
		if err != nil {
			return E.Cause(err, "create TLS config")
		}
	}
	tcpListener, err := s.listener.ListenTCP()
	if err != nil {
		return err
	}
	if s.tlsConfig != nil {
		if !common.Contains(s.tlsConfig.NextProtos(), http2.NextProtoTLS) {
			s.tlsConfig.SetNextProtos(append([]string{"h2"}, s.tlsConfig.NextProtos()...))
		}
		tcpListener = aTLS.NewListener(tcpListener, s.tlsConfig)
	}
	s.httpServer = &http.Server{
		Handler: chiRouter,
	}
	go func() {
		err = s.httpServer.Serve(tcpListener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("serve error: ", err)
		}
	}()
	return nil
}

func (s *Service) Close() error {
	return common.Close(
		common.PtrOrNil(s.httpServer),
		common.PtrOrNil(s.listener),
		s.tlsConfig,
	)
}
