package profiler

import (
	"context"
	"net/http"
	"net/http/pprof"
	runtimePprof "runtime/pprof"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxService "github.com/sagernet/sing-box/adapter/service"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/go-chi/chi/v5"
)

func RegisterService(registry *boxService.Registry) {
	boxService.Register[option.ProfilerServiceOptions](registry, C.TypeProfiler, NewService)
}

type Service struct {
	boxService.Adapter
	logger       log.ContextLogger
	listen       string
	readTimeout  time.Duration
	writeTimeout time.Duration
	server       *http.Server
}

func NewService(ctx context.Context, logger log.ContextLogger, tag string, options option.ProfilerServiceOptions) (adapter.Service, error) {
	if options.Listen == "" {
		return nil, E.New("missing listen")
	}
	return &Service{
		Adapter:      boxService.NewAdapter(C.TypeProfiler, tag),
		logger:       logger,
		listen:       options.Listen,
		readTimeout:  time.Duration(options.ReadTimeout),
		writeTimeout: time.Duration(options.WriteTimeout),
	}, nil
}

func (s *Service) Start(stage adapter.StartStage) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	r := chi.NewMux()
	r.Route("/debug/pprof", func(r chi.Router) {
		r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			if !strings.HasSuffix(req.URL.Path, "/") {
				http.Redirect(w, req, req.URL.Path+"/", http.StatusMovedPermanently)
				return
			}
			pprof.Index(w, req)
		})
		r.HandleFunc("/cmdline", pprof.Cmdline)
		r.HandleFunc("/profile", pprof.Profile)
		r.HandleFunc("/symbol", pprof.Symbol)
		r.HandleFunc("/trace", pprof.Trace)
		for _, p := range runtimePprof.Profiles() {
			name := p.Name()
			r.Handle("/"+name, pprof.Handler(name))
		}
		r.HandleFunc("/*", pprof.Index)
	})

	s.server = &http.Server{
		Addr:         s.listen,
		Handler:      r,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}
	s.logger.Info("profiler listening at ", s.listen)
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !E.IsClosed(err) {
			s.logger.Error(E.Cause(err, "serve profiler"))
		}
	}()
	return nil
}

func (s *Service) Close() error {
	if s.server != nil {
		_ = s.server.Close()
		s.server = nil
	}
	return nil
}
