import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function FilterEditorDialog() {
  return (
    <AppDialog
      className="filter-editor-dialog"
      title="Filter Builder"
      style={{ width: '600px', height: '400px', display: 'flex', flexDirection: 'column' }}
    >
      <div className="modal__header">
        <h3>Filter Builder</h3>
        <Button variant="ghost" size="icon-sm" className="modal__close">
          ×
        </Button>
      </div>
      <div className="modal__body" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        <div
          style={{
            flex: 1,
            border: '1px solid var(--border)',
            padding: '8px',
            marginBottom: '16px',
          }}
        >
          {/* Visual builder placeholder */}
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <NativeSelect defaultValue="and">
              <option value="and">AND</option>
              <option value="or">OR</option>
              <option value="not">NOT</option>
            </NativeSelect>
            <Button variant="outline">+</Button>
          </div>
          <div style={{ marginLeft: '16px', marginTop: '8px', display: 'flex', gap: '8px' }}>
            <Input
              type="text"
              placeholder="Attribute (e.g. objectClass)"
              defaultValue="objectClass"
            />
            <NativeSelect defaultValue="=">
              <option value="=">=</option>
              <option value=">=">&gt;=</option>
              <option value="<=">&lt;=</option>
              <option value="~=">~=</option>
            </NativeSelect>
            <Input type="text" placeholder="Value (e.g. *)" defaultValue="*" />
          </div>
        </div>
        <div>
          <Label>Raw RFC 4515 Filter</Label>
          <Textarea
            value="(&(objectClass=*))"
            readOnly
            style={{ width: '100%', height: '80px', fontFamily: 'monospace', marginTop: '8px' }}
          />
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Apply</Button>
      </div>
    </AppDialog>
  );
}
