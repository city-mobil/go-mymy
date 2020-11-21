package bridge

import (
	"errors"
	"fmt"

	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"

	"github.com/city-mobil/go-mymy/pkg/mymy"
)

var emptyGTID = mustCreateGTID(mysql.MySQLFlavor, "")

func mustCreateGTID(flavor, s string) mysql.GTIDSet {
	set, err := mysql.ParseGTIDSet(flavor, s)
	if err != nil {
		panic(err)
	}

	return set
}

type eventHandler struct {
	bridge   *Bridge
	gtidMode bool
}

func newEventHandler(b *Bridge, gtidMode bool) *eventHandler {
	return &eventHandler{
		bridge:   b,
		gtidMode: gtidMode,
	}
}

func (h *eventHandler) OnRotate(_ *replication.RotateEvent) error {
	return h.bridge.ctx.Err()
}

func (h *eventHandler) OnTableChanged(schema, table string) error {
	err := h.bridge.updateRule(schema, table)
	if err != nil && !errors.Is(err, ErrRuleNotExist) {
		return err
	}

	return nil
}

func (h *eventHandler) OnDDL(_ mysql.Position, _ *replication.QueryEvent) error {
	return h.bridge.ctx.Err()
}

func (h *eventHandler) OnXID(_ mysql.Position) error {
	return h.bridge.ctx.Err()
}

func (h *eventHandler) OnRow(e *canal.RowsEvent) error {
	rule, ok := h.bridge.rules[mymy.RuleKey(e.Table.Schema, e.Table.Name)]
	if !ok {
		return nil
	}

	queries, err := rule.Handler.OnRows(&mymy.RowsEvent{
		Action: mymy.Action(e.Action),
		Source: rule.Source,
		Rows:   e.Rows,
	})
	if err != nil {
		h.bridge.cancel()

		return fmt.Errorf("sync %s request, what: %w", e.Action, err)
	}

	h.bridge.syncCh <- batch(queries)

	return h.bridge.ctx.Err()
}

func (h *eventHandler) OnGTID(_ mysql.GTIDSet) error {
	return h.bridge.ctx.Err()
}

func (h *eventHandler) OnPosSynced(pos mysql.Position, set mysql.GTIDSet, force bool) error {
	if h.gtidMode {
		h.bridge.syncCh <- &savePos{
			pos:   newGTIDSet(set),
			force: force,
		}
	} else {
		h.bridge.syncCh <- &savePos{
			pos:   newBinlogPos(pos),
			force: force,
		}
	}

	return h.bridge.ctx.Err()
}

func (h *eventHandler) String() string {
	return "MyMyBridgeEventHandler"
}
