package bridge

import "errors"

func (b *Bridge) SupportsStoredACI() (bool, string) { return false, "not implemented" }
func (b *Bridge) ParseACI() error                   { return errors.New("not implemented") }
func (b *Bridge) RenderACI() error                  { return errors.New("not implemented") }
func (b *Bridge) ParseSubtreeSpec() error           { return errors.New("not implemented") }
func (b *Bridge) PreviewSubtreeSpec() error         { return errors.New("not implemented") }
