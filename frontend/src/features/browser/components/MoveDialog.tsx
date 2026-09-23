import React from 'react';

export function MoveDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal move-dialog">
        <div className="modal__header">
          <h3>Move Subtree</h3>
        </div>
        <div className="modal__body">
          <label>New Parent DN</label>
          <input type="text" placeholder="ou=newparent,dc=example,dc=org" style={{ width: '100%' }} />
          <div style={{ marginTop: '16px' }}>
            <label>Conflict Policy</label>
            <select style={{ width: '100%' }}>
              <option>Skip</option>
              <option>Overwrite</option>
            </select>
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Move</button>
        </div>
      </div>
    </div>
  );
}
