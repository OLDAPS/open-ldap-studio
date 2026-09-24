import React from 'react';

export function ResultBanner() {
  return (
    <div
      className="result-banner"
      style={{ backgroundColor: 'var(--bg-panel)', borderLeft: '4px solid green', padding: '8px' }}
    >
      <strong>Success</strong>: Entry modified (Result Code: 0)
    </div>
  );
}
