import React from 'react';

export function PendingEditsGuard() {
  return (
    <div className="modal-overlay">
      <div className="modal guard">
        <div className="modal__header">
          <h3>Unsaved Changes</h3>
        </div>
        <div className="modal__body">
          <p>You have unsaved edits on this entry. Navigating away will discard them.</p>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel Navigation</button>
          <button className="button button--danger">Discard Edits</button>
        </div>
      </div>
    </div>
  );
}
