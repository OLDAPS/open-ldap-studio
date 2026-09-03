import React from 'react';

export function ReferralChooser() {
  return (
    <div className="modal-overlay">
      <div className="modal referral-chooser-dialog">
        <div className="modal__header">
          <h3>Follow Referral</h3>
          <button className="modal__close">×</button>
        </div>
        <div className="modal__body">
          <p>The server referred you to <strong>ldap://other.example.org</strong>.</p>
          <p>Choose a connection profile to follow this referral:</p>
          <select style={{ width: '100%', marginTop: '8px' }}>
            <option>Localhost OpenLDAP</option>
            <option>Create New Profile...</option>
          </select>
          <div style={{ marginTop: '16px' }}>
            <label>
              <input type="checkbox" /> Remember for this session
            </label>
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Follow</button>
        </div>
      </div>
    </div>
  );
}
