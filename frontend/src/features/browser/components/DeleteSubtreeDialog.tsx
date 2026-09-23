import React from 'react';

export function DeleteSubtreeDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal delete-subtree">
        <div className="modal__header">
          <h3>Delete Subtree</h3>
        </div>
        <div className="modal__body">
          <p>You are about to delete an entry and <strong>45</strong> children.</p>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--danger">Delete Subtree</button>
        </div>
      </div>
    </div>
  );
}
