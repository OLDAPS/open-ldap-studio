import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function RecordDetail() {
  return (
    <AppDialog className="record-detail" title="Record Detail" style={{ width: '600px' }}>
      <div className="modal__header">
        <h3>Record Detail</h3>
      </div>
      <div className="modal__body">
        <p>
          <strong>Operation:</strong> Modify
        </p>
        <pre>before: description: Old</pre>
        <pre>after: description: New</pre>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Export LDIF</Button>
        <Button variant="destructive">Reverse</Button>
        <Button variant="default">Replay</Button>
      </div>
    </AppDialog>
  );
}
