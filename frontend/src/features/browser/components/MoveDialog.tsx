import { Label } from '@/components/ui/label';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function MoveDialog() {
  return (
    <AppDialog className="move-dialog" title="Move Subtree">
      <div className="modal__header">
        <h3>Move Subtree</h3>
      </div>
      <div className="modal__body">
        <Label>New Parent DN</Label>
        <Input type="text" placeholder="ou=newparent,dc=example,dc=org" style={{ width: '100%' }} />
        <div style={{ marginTop: '16px' }}>
          <Label>Conflict Policy</Label>
          <NativeSelect style={{ width: '100%' }}>
            <option>Skip</option>
            <option>Overwrite</option>
          </NativeSelect>
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Move</Button>
      </div>
    </AppDialog>
  );
}
