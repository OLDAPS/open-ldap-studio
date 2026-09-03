package bridge
import "errors"
func (b *Bridge) ReadValue() error { return errors.New("not implemented") }
func (b *Bridge) DecodeValue() error { return errors.New("not implemented") }
func (b *Bridge) EncodeValue() error { return errors.New("not implemented") }
func (b *Bridge) LoadValueFromFile() error { return errors.New("not implemented") }
func (b *Bridge) SaveValueToFile() error { return errors.New("not implemented") }
