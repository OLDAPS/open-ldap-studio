import React from 'react';

export function DiffSummary() {
  return (
    <div className="diff-summary" style={{ padding: '16px' }}>
      <h4>Differences (2 found)</h4>
      <ul>
        <li>cn=admin - Missing in Target</li>
        <li>cn=user - Attribute mismatch (mail)</li>
      </ul>
    </div>
  );
}
