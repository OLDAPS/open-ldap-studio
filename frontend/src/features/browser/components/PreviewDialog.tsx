import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function PreviewDialog() {
  return (
    <AppDialog className="preview-dialog" title="Preview Changes">
      <div className="modal__header">
        <h3>Preview Changes</h3>
      </div>
      <div className="modal__body">
        <p>You are about to modify 1 entry.</p>
        <pre>replace: description\ndescription: Admin user\n-\n</pre>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Commit</Button>
      </div>
    </AppDialog>
  );
}
