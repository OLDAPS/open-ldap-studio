import React from 'react';

export function RenameDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal rename-dialog">
        <div className="modal__header">
          <h3>Rename Entry</h3>
        </div>
        <div className="modal__body">
          <label>New RDN</label>
          <input type="text" placeholder="cn=newname" style={{ width: '100%' }} />
          <div style={{ marginTop: '16px' }}>
            <label><input type="checkbox" defaultChecked /> Delete old RDN from entry</label>
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Rename</button>
        </div>
      </div>
    </div>
  );
}
