package api

import (
	"context"
)

type Sentinel interface {
	WatchStop(context.Context) error
	WatchStart(context.Context) error
}
