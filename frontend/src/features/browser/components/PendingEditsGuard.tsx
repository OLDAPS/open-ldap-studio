import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function PendingEditsGuard() {
  return (
    <AppDialog className="guard" title="Unsaved Changes">
      <div className="modal__header">
        <h3>Unsaved Changes</h3>
      </div>
      <div className="modal__body">
        <p>You have unsaved edits on this entry. Navigating away will discard them.</p>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel Navigation</Button>
        <Button variant="destructive">Discard Edits</Button>
      </div>
    </AppDialog>
  );
}
