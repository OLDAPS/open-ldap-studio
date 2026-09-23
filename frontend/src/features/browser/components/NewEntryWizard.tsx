import React from 'react';

export function NewEntryWizard() {
  return (
    <div className="modal-overlay">
      <div className="modal new-entry-wizard">
        <div className="modal__header">
          <h3>New Entry</h3>
        </div>
        <div className="modal__body">
          <label>Object Classes</label>
          <input type="text" placeholder="inetOrgPerson" />
          <label>RDN</label>
          <input type="text" placeholder="cn=newuser" />
        </div>
        <div className="modal__footer">
          <button className="button button--primary">Next</button>
        </div>
      </div>
    </div>
  );
}
