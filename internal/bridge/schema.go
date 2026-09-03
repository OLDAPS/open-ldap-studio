package bridge
import "errors"
func (b *Bridge) ReadSchema() error { return errors.New("not implemented") }
func (b *Bridge) SchemaAvailable() bool { return false }
func (b *Bridge) CompareSchemas() error { return errors.New("not implemented") }
