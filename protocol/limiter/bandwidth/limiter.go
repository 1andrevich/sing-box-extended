package bandwidth

import (
	"context"
)

type Limiter interface {
	WaitN(ctx context.Context, n int) (err error)
}
