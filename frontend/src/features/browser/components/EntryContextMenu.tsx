import React from 'react';

export function EntryContextMenu() {
  return (
    <div className="context-menu" style={{ position: 'absolute', backgroundColor: 'var(--bg-panel)', border: '1px solid var(--border)', padding: '4px' }}>
      <div className="menu-item">New Entry</div>
      <div className="menu-item">Rename</div>
      <div className="menu-item">Delete</div>
    </div>
  );
}
