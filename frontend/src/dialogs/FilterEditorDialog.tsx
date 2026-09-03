import React from 'react';

export function FilterEditorDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal filter-editor-dialog" style={{ width: '600px', height: '400px', display: 'flex', flexDirection: 'column' }}>
        <div className="modal__header">
          <h3>Filter Builder</h3>
          <button className="modal__close">×</button>
        </div>
        <div className="modal__body" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1, border: '1px solid var(--border)', padding: '8px', marginBottom: '16px' }}>
            {/* Visual builder placeholder */}
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <select defaultValue="and">
                <option value="and">AND</option>
                <option value="or">OR</option>
                <option value="not">NOT</option>
              </select>
              <button className="button">+</button>
            </div>
            <div style={{ marginLeft: '16px', marginTop: '8px', display: 'flex', gap: '8px' }}>
              <input type="text" placeholder="Attribute (e.g. objectClass)" defaultValue="objectClass" />
              <select defaultValue="=">
                <option value="=">=</option>
                <option value=">=">&gt;=</option>
                <option value="<=">&lt;=</option>
                <option value="~=">~=</option>
              </select>
              <input type="text" placeholder="Value (e.g. *)" defaultValue="*" />
            </div>
          </div>
          <div>
            <label>Raw RFC 4515 Filter</label>
            <textarea 
              value="(&(objectClass=*))" 
              readOnly 
              style={{ width: '100%', height: '80px', fontFamily: 'monospace', marginTop: '8px' }} 
            />
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">Apply</button>
        </div>
      </div>
    </div>
  );
}
