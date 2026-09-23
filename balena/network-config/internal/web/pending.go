package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/normalframework/netcfgd/internal/netmgr"
)

// confirmWindow is how long an interface change waits for confirmation before
// it is rolled back. Long enough for a DHCP lease and a page reload, short
// enough that a locked-out operator is not left waiting.
const confirmWindow = 90 * time.Second

// armPending applies the rollback guard around an interface change. The caller
// makes the change; this arranges for it to be undone unless confirmed.
func (a *App) armPending(iface string, snap netmgr.Snapshot) (string, error) {
	tokenBytes := make([]byte, 12)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate confirmation token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	ctx, cancel := context.WithCancel(context.Background())
	pc := &pendingChange{
		Iface:    iface,
		Snapshot: snap,
		Deadline: time.Now().Add(confirmWindow),
		Token:    token,
		cancel:   cancel,
	}

	a.mu.Lock()
	// Only one change can be in flight; a new one supersedes any older guard.
	if a.pending != nil {
		a.pending.cancel()
	}
	a.pending = pc
	a.mu.Unlock()

	go a.watchPending(ctx, pc)
	return token, nil
}

// watchPending rolls the change back when the window closes.
func (a *App) watchPending(ctx context.Context, pc *pendingChange) {
	select {
	case <-ctx.Done():
		// Confirmed or superseded; nothing to do.
		return
	case <-time.After(time.Until(pc.Deadline)):
	}

	a.mu.Lock()
	if a.pending != pc {
		a.mu.Unlock()
		return
	}
	a.pending = nil
	a.mu.Unlock()

	a.log.Warnf("web", "system",
		fmt.Sprintf("reverting the change to %s", pc.Iface),
		fmt.Sprintf("not confirmed within %s", confirmWindow))

	if a.nm == nil {
		return
	}
	if err := a.nm.Restore(pc.Snapshot); err != nil {
		a.log.Errorf("web", "system", "could not revert "+pc.Iface, err.Error())
		return
	}
	a.log.Changef("web", "system", "reverted "+pc.Iface+" to its previous settings", "")
}

// currentPending returns the change awaiting confirmation, if any.
func (a *App) currentPending() *pendingChange {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending != nil && time.Now().After(a.pending.Deadline) {
		return nil
	}
	return a.pending
}

// confirmPending keeps a change, cancelling the rollback.
func (a *App) confirmPending(token string) error {
	a.mu.Lock()
	pc := a.pending
	if pc == nil {
		a.mu.Unlock()
		return fmt.Errorf("there is no change waiting for confirmation")
	}
	if token != "" && token != pc.Token {
		a.mu.Unlock()
		return fmt.Errorf("that confirmation is for a different change")
	}
	a.pending = nil
	a.mu.Unlock()

	pc.cancel()
	a.log.Changef("web", "", "confirmed the change to "+pc.Iface, "")
	return nil
}

// revertPending rolls a change back immediately rather than waiting.
func (a *App) revertPending() error {
	a.mu.Lock()
	pc := a.pending
	a.pending = nil
	a.mu.Unlock()

	if pc == nil {
		return fmt.Errorf("there is no change waiting for confirmation")
	}
	pc.cancel()

	if a.nm == nil {
		return fmt.Errorf("NetworkManager is not reachable, so the change cannot be reverted")
	}
	if err := a.nm.Restore(pc.Snapshot); err != nil {
		return fmt.Errorf("could not revert %s: %w", pc.Iface, err)
	}
	a.log.Changef("web", "", "reverted "+pc.Iface+" on request", "")
	return nil
}
