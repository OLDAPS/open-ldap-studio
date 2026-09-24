import { Button } from '@/components/ui/button';
import React from 'react';

export function OperationQueue() {
  return (
    <div className="operation-queue" style={{ padding: '16px', border: '1px solid var(--border)' }}>
      <h4>Operation Queue</h4>
      <div>Copying 500 entries... (50%)</div>
      <Button variant="outline">Pause</Button>
      <Button variant="destructive" style={{ marginLeft: '8px' }}>
        Abort
      </Button>
    </div>
  );
}
