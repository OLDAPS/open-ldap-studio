import { Label } from '@/components/ui/label';
import { NativeSelect } from '@/components/ui/native-select';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { AppDialog } from '@/components/ui/app-dialog';
import React, { useState } from 'react';

export function ImportExportWizard() {
  const [mode, setMode] = useState<'import' | 'export'>('export');

  return (
    <AppDialog className="import-export-wizard" title="Data Exchange">
      <div className="modal__header">
        <h3>Data Exchange</h3>
      </div>
      <div className="modal__body">
        <RadioGroup
          value={mode}
          onValueChange={(value) => setMode(value as 'import' | 'export')}
          style={{ marginBottom: '16px' }}
        >
          <Label style={{ marginRight: '16px' }}>
            <RadioGroupItem value="export" /> Export
          </Label>
          <Label>
            <RadioGroupItem value="import" /> Import
          </Label>
        </RadioGroup>
        {mode === 'export' && (
          <div>
            <Label>Format</Label>
            <NativeSelect style={{ width: '100%', marginBottom: '8px' }}>
              <option>LDIF</option>
              <option>DSML v2</option>
              <option>CSV</option>
              <option>JSON</option>
              <option>XLSX</option>
              <option>ODS</option>
            </NativeSelect>
          </div>
        )}
        {mode === 'import' && (
          <div>
            <Label>Source File</Label>
            <Input type="file" style={{ width: '100%', marginBottom: '8px' }} />
          </div>
        )}
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">{mode === 'export' ? 'Start Export' : 'Start Import'}</Button>
      </div>
    </AppDialog>
  );
}
