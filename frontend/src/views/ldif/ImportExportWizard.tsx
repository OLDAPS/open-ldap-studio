import React, { useState } from 'react';

export function ImportExportWizard() {
  const [mode, setMode] = useState<'import' | 'export'>('export');
  
  return (
    <div className="modal-overlay">
      <div className="modal import-export-wizard">
        <div className="modal__header">
          <h3>Data Exchange</h3>
        </div>
        <div className="modal__body">
          <div style={{ marginBottom: '16px' }}>
            <label style={{ marginRight: '16px' }}><input type="radio" checked={mode === 'export'} onChange={() => setMode('export')} /> Export</label>
            <label><input type="radio" checked={mode === 'import'} onChange={() => setMode('import')} /> Import</label>
          </div>
          {mode === 'export' && (
            <div>
              <label>Format</label>
              <select style={{ width: '100%', marginBottom: '8px' }}>
                <option>LDIF</option>
                <option>DSML v2</option>
                <option>CSV</option>
                <option>JSON</option>
                <option>XLSX</option>
                <option>ODS</option>
              </select>
            </div>
          )}
          {mode === 'import' && (
            <div>
              <label>Source File</label>
              <input type="file" style={{ width: '100%', marginBottom: '8px' }} />
            </div>
          )}
        </div>
        <div className="modal__footer">
          <button className="button">Cancel</button>
          <button className="button button--primary">{mode === 'export' ? 'Start Export' : 'Start Import'}</button>
        </div>
      </div>
    </div>
  );
}
