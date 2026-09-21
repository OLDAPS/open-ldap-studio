import React from 'react';

export function RecordDetail() {
  return (
    <div className="modal-overlay">
      <div className="modal record-detail" style={{ width: '600px' }}>
        <div className="modal__header">
          <h3>Record Detail</h3>
        </div>
        <div className="modal__body">
          <p><strong>Operation:</strong> Modify</p>
          <pre>before: description: Old</pre>
          <pre>after: description: New</pre>
        </div>
        <div className="modal__footer">
          <button className="button">Export LDIF</button>
          <button className="button button--danger">Reverse</button>
          <button className="button button--primary">Replay</button>
        </div>
      </div>
    </div>
  );
}
