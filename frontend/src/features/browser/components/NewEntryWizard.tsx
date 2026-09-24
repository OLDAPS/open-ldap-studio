import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function NewEntryWizard() {
  return (
    <AppDialog className="new-entry-wizard" title="New Entry">
      <div className="modal__header">
        <h3>New Entry</h3>
      </div>
      <div className="modal__body">
        <Label>Object Classes</Label>
        <Input type="text" placeholder="inetOrgPerson" />
        <Label>RDN</Label>
        <Input type="text" placeholder="cn=newuser" />
      </div>
      <div className="modal__footer">
        <Button variant="default">Next</Button>
      </div>
    </AppDialog>
  );
}
