import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function RenameDialog() {
  return (
    <AppDialog className="rename-dialog" title="Rename Entry">
      <div className="modal__header">
        <h3>Rename Entry</h3>
      </div>
      <div className="modal__body">
        <Label>New RDN</Label>
        <Input type="text" placeholder="cn=newname" style={{ width: '100%' }} />
        <div style={{ marginTop: '16px' }}>
          <Label>
            <Checkbox defaultChecked /> Delete old RDN from entry
          </Label>
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Rename</Button>
      </div>
    </AppDialog>
  );
}
