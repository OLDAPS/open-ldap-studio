import React from 'react';

export function CopyMoveDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal copy-move-dialog">
        <div className="modal__header">
          <h3>Copy / Move</h3>
        </div>
        <div className="modal__body">
          <label><input type="radio" name="mode" defaultChecked /> Copy</label>
          <label><input type="radio" name="mode" /> Copy Subtree</label>
          <label><input type="radio" name="mode" /> Move</label>
          
          <div style={{ marginTop: '16px' }}>
            <label>Conflict Policy</label>
            <select style={{ width: '100%' }}>
              <option>Skip</option>
              <option>Overwrite</option>
              <option>Merge</option>
              <option>Rename</option>
            </select>
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Execute</button>
        </div>
      </div>
    </div>
  );
}
