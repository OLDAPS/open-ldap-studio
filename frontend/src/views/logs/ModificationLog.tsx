import React from 'react';
import { HistoryFilter } from './HistoryFilter';

export function ModificationLog() {
  return (
    <div className="modification-log" style={{ display: 'flex', height: '100%' }}>
      <div className="sidebar" style={{ width: '250px', borderRight: '1px solid var(--border)', padding: '16px' }}>
        <HistoryFilter />
      </div>
      <div className="main-content" style={{ flex: 1, padding: '16px', display: 'flex', flexDirection: 'column' }}>
        <table style={{ width: '100%', textAlign: 'left' }}>
          <thead>
            <tr><th>Time</th><th>Operation</th><th>Target</th><th>Result</th></tr>
          </thead>
          <tbody>
            <tr>
              <td>12:00:01</td>
              <td>Modify</td>
              <td>cn=admin</td>
              <td style={{ color: 'green' }}>Success (0)</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}
