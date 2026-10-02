import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function DeleteSubtreeDialog() {
  return (
    <AppDialog className="delete-subtree" title="Delete Subtree">
      <div className="modal__header">
        <h3>Delete Subtree</h3>
      </div>
      <div className="modal__body">
        <p>
          You are about to delete an entry and <strong>45</strong> children.
        </p>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="destructive">Delete Subtree</Button>
      </div>
    </AppDialog>
  );
}
