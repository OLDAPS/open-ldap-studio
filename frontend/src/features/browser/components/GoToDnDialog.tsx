import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function GoToDnDialog() {
  return (
    <AppDialog className="goto-dn-dialog" title="Go to DN">
      <div className="modal__header">
        <h3>Go to DN</h3>
        <Button variant="ghost" size="icon-sm" className="modal__close">
          ×
        </Button>
      </div>
      <div className="modal__body">
        <Label>Enter Distinguished Name or LDAP URL:</Label>
        <Input
          type="text"
          placeholder="cn=admin,dc=example,dc=org"
          style={{ width: '100%', marginTop: '8px' }}
        />
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Go</Button>
      </div>
    </AppDialog>
  );
}
