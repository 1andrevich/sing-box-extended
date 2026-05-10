package fallback

import (
	"context"

	mDNS "github.com/miekg/dns"
	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
)

type ExchangeStrategy = func(ctx context.Context, message *mDNS.Msg) (*mDNS.Msg, error)

func parallelStrategy(servers []adapter.DNSTransport, logger logger.ContextLogger) ExchangeStrategy {
	return func(ctx context.Context, message *mDNS.Msg) (*mDNS.Msg, error) {
		queryCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		type result struct {
			response *mDNS.Msg
			err      error
		}
		results := make(chan result)
		for _, server := range servers {
			go func() {
				response, err := server.Exchange(queryCtx, message)
				select {
				case results <- result{response, err}:
				case <-queryCtx.Done():
				}
			}()
		}
		var lastErr error
		for range servers {
			select {
			case result := <-results:
				if result.err != nil {
					lastErr = result.err
					continue
				}
				return result.response, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return nil, lastErr
	}
}

func sequentialStrategy(servers []adapter.DNSTransport, logger logger.ContextLogger) ExchangeStrategy {
	return func(ctx context.Context, message *mDNS.Msg) (*mDNS.Msg, error) {
		var lastErr error
		for _, server := range servers {
			response, err := server.Exchange(ctx, message)
			if err != nil {
				lastErr = err
				continue
			}
			return response, nil
		}
		return nil, lastErr
	}
}

func CreateStrategy(strategy string, servers []adapter.DNSTransport, logger logger.ContextLogger) (ExchangeStrategy, error) {
	switch strategy {
	case "parallel":
		return parallelStrategy(servers, logger), nil
	case "", "sequential":
		return sequentialStrategy(servers, logger), nil
	default:
		return nil, E.New("strategy not found: ", strategy)
	}
}
