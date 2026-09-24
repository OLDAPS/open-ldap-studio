import { Button } from '@/components/ui/button';
import React from 'react';

export function ProgressView() {
  return (
    <div
      className="progress-view"
      style={{
        position: 'fixed',
        bottom: 0,
        right: 0,
        padding: '16px',
        borderTop: '1px solid var(--border)',
        borderLeft: '1px solid var(--border)',
        backgroundColor: 'var(--bg-panel)',
      }}
    >
      <h4>Active Tasks</h4>
      <div>Search: 10,000 entries (running)</div>
      <Button variant="outline">Cancel</Button>
    </div>
  );
}
