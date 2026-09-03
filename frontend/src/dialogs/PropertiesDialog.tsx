import React from 'react';

export function PropertiesDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal properties-dialog">
        <div className="modal__header">
          <h3>Properties</h3>
          <button className="modal__close">×</button>
        </div>
        <div className="modal__body">
          <table style={{ width: '100%', textAlign: 'left' }}>
            <tbody>
              <tr>
                <th style={{ width: '150px' }}>Type</th>
                <td>Entry</td>
              </tr>
              <tr>
                <th>DN</th>
                <td style={{ fontFamily: 'monospace' }}>cn=admin,dc=example,dc=org</td>
              </tr>
              <tr>
                <th>Size</th>
                <td>1.2 KB</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
