import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function AttributeWizard() {
  return (
    <AppDialog className="attribute-wizard" title="Add Attribute">
      <div className="modal__header">
        <h3>Add Attribute</h3>
      </div>
      <div className="modal__body">
        <Input type="text" placeholder="Attribute name" />
        <Label>
          <Checkbox /> ;binary
        </Label>
        <Label>
          <Checkbox /> ;lang-de
        </Label>
      </div>
      <div className="modal__footer">
        <Button variant="default">Add</Button>
      </div>
    </AppDialog>
  );
}
