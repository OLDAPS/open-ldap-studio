import React from 'react';

export function ReconcilePanel() {
  return (
    <div className="reconcile-panel" style={{ padding: '16px' }}>
      <h4>Reconcile</h4>
      <p>Generate LDIF to apply source state to target.</p>
      <button className="button button--primary" style={{ width: '100%', marginBottom: '8px' }}>Dry Run</button>
      <button className="button button--danger" style={{ width: '100%' }}>Apply Changes</button>
    </div>
  );
}
