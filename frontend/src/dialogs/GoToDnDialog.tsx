import React from 'react';

export function GoToDnDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal goto-dn-dialog">
        <div className="modal__header">
          <h3>Go to DN</h3>
          <button className="modal__close">×</button>
        </div>
        <div className="modal__body">
          <label>Enter Distinguished Name or LDAP URL:</label>
          <input type="text" placeholder="cn=admin,dc=example,dc=org" style={{ width: '100%', marginTop: '8px' }} />
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Go</button>
        </div>
      </div>
    </div>
  );
}
