import { Button } from '@/components/ui/button';
import React from 'react';
import { DiffSummary } from './DiffSummary';
import { ReconcilePanel } from './ReconcilePanel';

export function CompareView() {
  return (
    <div
      className="compare-view"
      style={{ display: 'flex', height: '100%', flexDirection: 'column' }}
    >
      <div className="header" style={{ padding: '16px', borderBottom: '1px solid var(--border)' }}>
        <h3>Directory Comparison</h3>
        <Button variant="outline">Source: Localhost OpenLDAP</Button>
        <Button variant="outline" style={{ marginLeft: '8px' }}>
          Target: Prod Server
        </Button>
      </div>
      <div style={{ display: 'flex', flex: 1 }}>
        <div style={{ flex: 1, borderRight: '1px solid var(--border)' }}>
          <DiffSummary />
        </div>
        <div style={{ width: '300px' }}>
          <ReconcilePanel />
        </div>
      </div>
    </div>
  );
}
