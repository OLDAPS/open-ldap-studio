package bridge

import "errors"

func (b *Bridge) HashPassword() error          { return errors.New("not implemented") }
func (b *Bridge) VerifyPassword() error        { return errors.New("not implemented") }
func (b *Bridge) PasswordModify() error        { return errors.New("not implemented") }
func (b *Bridge) SupportsPasswordModify() bool { return false }
