import { Table, TableBody, TableCell, TableHead, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function PropertiesDialog() {
  return (
    <AppDialog className="properties-dialog" title="Properties">
      <div className="modal__header">
        <h3>Properties</h3>
        <Button variant="ghost" size="icon-sm" className="modal__close">
          ×
        </Button>
      </div>
      <div className="modal__body">
        <Table style={{ width: '100%', textAlign: 'left' }}>
          <TableBody>
            <TableRow>
              <TableHead style={{ width: '150px' }}>Type</TableHead>
              <TableCell>Entry</TableCell>
            </TableRow>
            <TableRow>
              <TableHead>DN</TableHead>
              <TableCell style={{ fontFamily: 'monospace' }}>cn=admin,dc=example,dc=org</TableCell>
            </TableRow>
            <TableRow>
              <TableHead>Size</TableHead>
              <TableCell>1.2 KB</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </AppDialog>
  );
}
