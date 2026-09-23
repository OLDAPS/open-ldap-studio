import React from 'react';

export function HistoryFilter() {
  return (
    <div className="history-filter">
      <h4>Filter History</h4>
      <input type="text" placeholder="DN or Subtree..." style={{ width: '100%', marginBottom: '8px' }} />
      <select style={{ width: '100%', marginBottom: '8px' }}>
        <option>All Operations</option>
        <option>Modify</option>
        <option>Add</option>
        <option>Delete</option>
      </select>
    </div>
  );
}
