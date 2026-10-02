import { Label } from '@/components/ui/label';
import { NativeSelect } from '@/components/ui/native-select';
import { Button } from '@/components/ui/button';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function CopyMoveDialog() {
  return (
    <AppDialog className="copy-move-dialog" title="Copy or Move">
      <div className="modal__header">
        <h3>Copy / Move</h3>
      </div>
      <div className="modal__body">
        <RadioGroup defaultValue="copy">
          <Label>
            <RadioGroupItem value="copy" /> Copy
          </Label>
          <Label>
            <RadioGroupItem value="copy-subtree" /> Copy Subtree
          </Label>
          <Label>
            <RadioGroupItem value="move" /> Move
          </Label>
        </RadioGroup>

        <div style={{ marginTop: '16px' }}>
          <Label>Conflict Policy</Label>
          <NativeSelect style={{ width: '100%' }}>
            <option>Skip</option>
            <option>Overwrite</option>
            <option>Merge</option>
            <option>Rename</option>
          </NativeSelect>
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Execute</Button>
      </div>
    </AppDialog>
  );
}
