import React from 'react';

export function AttributeWizard() {
  return (
    <div className="modal-overlay">
      <div className="modal attribute-wizard">
        <div className="modal__header">
          <h3>Add Attribute</h3>
        </div>
        <div className="modal__body">
          <input type="text" placeholder="Attribute name" />
          <label><input type="checkbox" /> ;binary</label>
          <label><input type="checkbox" /> ;lang-de</label>
        </div>
        <div className="modal__footer">
          <button className="button button--primary">Add</button>
        </div>
      </div>
    </div>
  );
}
