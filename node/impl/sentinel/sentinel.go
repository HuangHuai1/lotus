package sentinel

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/events"
	"github.com/filecoin-project/lotus/sentinel"
)

var log = logging.Logger("sentinel-module")

type SentinelAPI struct {
	fx.In

	Events *events.Events
}

func (m *SentinelAPI) WatchStart(ctx context.Context) error {
	log.Info("starting sentinel watch")
	return m.Events.Observe(&sentinel.LoggingTipSetObserver{})
}

func (m *SentinelAPI) WatchStop(ctx context.Context) error {
	log.Info("stopping sentinel watch")
	return nil
}

// SentinelUnavailable is an implementation of the sentinel api that returns an unavailable error for every request
type SentinelUnavailable struct {
	fx.In
}

func (SentinelUnavailable) WatchStart(ctx context.Context) error {
	return xerrors.Errorf("sentinel unavailable")
}

func (SentinelUnavailable) WatchStop(ctx context.Context) error {
	return xerrors.Errorf("sentinel unavailable")
}

var _ api.Sentinel = &SentinelAPI{}
