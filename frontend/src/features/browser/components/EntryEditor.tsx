import React, { useState } from 'react';

export function EntryEditor() {
  const [edits, setEdits] = useState(false);
  
  return (
    <div className="entry-editor">
      <h4>Entry Editor</h4>
      <div style={{ fontSize: '0.8rem', color: 'var(--accent)', marginBottom: '8px' }}>
        Schema active: 4 required attributes, 12 permitted
      </div>
      <input type="text" defaultValue="cn=admin,dc=example,dc=org" onChange={() => setEdits(true)} />
      {edits && <button className="button button--primary" style={{ marginLeft: '8px' }}>Preview Changes</button>}
    </div>
  );
}
