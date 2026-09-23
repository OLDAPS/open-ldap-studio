import React from 'react';

export function OperationQueue() {
  return (
    <div className="operation-queue" style={{ padding: '16px', border: '1px solid var(--border)' }}>
      <h4>Operation Queue</h4>
      <div>Copying 500 entries... (50%)</div>
      <button className="button">Pause</button>
      <button className="button button--danger" style={{ marginLeft: '8px' }}>Abort</button>
    </div>
  );
}
