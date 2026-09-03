package bridge
import "errors"
func (b *Bridge) ReversalAvailable() (bool, string) { return false, "not implemented" }
func (b *Bridge) History() error { return errors.New("not implemented") }
func (b *Bridge) ExportHistoryRecord() error { return errors.New("not implemented") }
func (b *Bridge) PrepareReplay() error { return errors.New("not implemented") }
func (b *Bridge) PrepareReversal() error { return errors.New("not implemented") }
