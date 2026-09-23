import React from 'react';

export function PreviewDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal preview-dialog">
        <div className="modal__header">
          <h3>Preview Changes</h3>
        </div>
        <div className="modal__body">
          <p>You are about to modify 1 entry.</p>
          <pre>replace: description\ndescription: Admin user\n-\n</pre>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Commit</button>
        </div>
      </div>
    </div>
  );
}
