import { Button } from '@/components/ui/button';
import React from 'react';

export function ReconcilePanel() {
  return (
    <div className="reconcile-panel" style={{ padding: '16px' }}>
      <h4>Reconcile</h4>
      <p>Generate LDIF to apply source state to target.</p>
      <Button variant="default" style={{ width: '100%', marginBottom: '8px' }}>
        Dry Run
      </Button>
      <Button variant="destructive" style={{ width: '100%' }}>
        Apply Changes
      </Button>
    </div>
  );
}
