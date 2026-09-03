package bridge
import "errors"
func (b *Bridge) ConfigAvailable() bool { return false }
func (b *Bridge) ReadConfigTree() error { return errors.New("not implemented") }
