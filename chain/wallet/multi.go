package wallet

import (
	"context"

	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	ledgerwallet "github.com/filecoin-project/lotus/chain/wallet/ledger"
	"github.com/filecoin-project/lotus/chain/wallet/remotewallet"
)

type MultiWallet struct {
	fx.In // "constructed" with fx.In instead of normal constructor

	Local  *LocalWallet               `optional:"true"`
	Remote *remotewallet.RemoteWallet `optional:"true"`
	Ledger *ledgerwallet.LedgerWallet `optional:"true"`
}

type getif interface {
	api.WalletAPI

	// workaround for the fact that iface(*struct(nil)) != nil
	Get() api.WalletAPI
}

func firstNonNil(wallets ...getif) api.WalletAPI {
	for _, w := range wallets {
		if w.Get() != nil {
			return w
		}
	}

	return nil
}

func nonNil(wallets ...getif) []api.WalletAPI {
	var out []api.WalletAPI
	for _, w := range wallets {
		if w.Get() == nil {
			continue
		}

		out = append(out, w)
	}

	return out
}

func (m MultiWallet) find(ctx context.Context, address address.Address, wallets ...getif) (api.WalletAPI, error) {
	ws := nonNil(wallets...)

	for _, w := range ws {
		have, err := w.WalletHas(ctx, address)
		if err != nil {
			return nil, err
		}

		if have {
			return w, nil
		}
	}

	return nil, nil
}

func (m MultiWallet) WalletNew(ctx context.Context, keyType types.KeyType) (address.Address, error) {
	var local getif = m.Local
	if keyType == types.KTSecp256k1Ledger {
		local = m.Ledger
	}

	w := firstNonNil(m.Remote, local)
	if w == nil {
		return address.Undef, xerrors.Errorf("no wallet backends supporting key type: %s", keyType)
	}

	return w.WalletNew(ctx, keyType)
}

func (m MultiWallet) WalletHas(ctx context.Context, address address.Address) (bool, error) {
	w, err := m.find(ctx, address, m.Remote, m.Ledger, m.Local)
	return w != nil, err
}

func (m MultiWallet) WalletList(ctx context.Context) ([]address.Address, error) {
	out := make([]address.Address, 0)
	seen := map[address.Address]struct{}{}

	ws := nonNil(m.Remote, m.Ledger, m.Local)
	for _, w := range ws {
		l, err := w.WalletList(ctx)
		if err != nil {
			return nil, err
		}

		for _, a := range l {
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}

			out = append(out, a)
		}
	}

	return out, nil
}

func (m MultiWallet) WalletSign(ctx context.Context, signer address.Address, toSign []byte, meta api.MsgMeta) (*crypto.Signature, error) {
	w, err := m.find(ctx, signer, m.Remote, m.Ledger, m.Local)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, xerrors.Errorf("key not found")
	}

	return w.WalletSign(ctx, signer, toSign, meta)
}

func (m MultiWallet) WalletExport(ctx context.Context, address address.Address, password string) (*types.KeyInfo, error) {
	w, err := m.find(ctx, address, m.Remote, m.Local)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, xerrors.Errorf("key not found")
	}

	return w.WalletExport(ctx, address, password)
}

func (m MultiWallet) WalletImport(ctx context.Context, info *types.KeyInfo) (address.Address, error) {
	var local getif = m.Local
	if info.Type == types.KTSecp256k1Ledger {
		local = m.Ledger
	}

	w := firstNonNil(m.Remote, local)
	if w == nil {
		return address.Undef, xerrors.Errorf("no wallet backends configured")
	}

	return w.WalletImport(ctx, info)
}

func (m MultiWallet) WalletDelete(ctx context.Context, address address.Address, pass string) error {
	for {
		w, err := m.find(ctx, address, m.Remote, m.Ledger, m.Local)
		if err != nil {
			return err
		}
		if w == nil {
			return nil
		}

		if err := w.WalletDelete(ctx, address, pass); err != nil {
			return err
		}
	}
}

func (m MultiWallet) WalletChangePasswd(ctx context.Context, newPasswd string) (bool, error) {
	return m.Local.WalletChangePasswd(ctx, newPasswd)
}

func (m MultiWallet) DeleteKey2(addr address.Address) error {
	return m.Local.DeleteKey2(addr)
}

func (m MultiWallet) WalletClearPasswd(ctx context.Context) (bool, error) {
	return m.Local.WalletClearPasswd(ctx)
}

// WalletIsLock
func (m MultiWallet) WalletIsLock(ctx context.Context) (bool, error) {
	return m.Local.WalletIsLock(ctx)
}

func (m MultiWallet) WalletLock(ctx context.Context) error {
	return m.Local.WalletLock(ctx)
}

// // WalletUnlock
func (m MultiWallet) WalletUnlock(ctx context.Context, password string) error {
	return m.Local.WalletUnlock(ctx, password)
}

func (m MultiWallet) WalletSignMessage2(ctx context.Context, k address.Address, msg *types.Message, passwd string) (*types.SignedMessage, error) {
	return m.Local.WalletSignMessage2(ctx, k, msg, passwd)
}

var _ api.WalletAPI = MultiWallet{}
