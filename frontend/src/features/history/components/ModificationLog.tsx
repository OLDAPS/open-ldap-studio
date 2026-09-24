import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import React from 'react';
import { HistoryFilter } from './HistoryFilter';

export function ModificationLog() {
  return (
    <div className="modification-log" style={{ display: 'flex', height: '100%' }}>
      <div
        className="sidebar"
        style={{ width: '250px', borderRight: '1px solid var(--border)', padding: '16px' }}
      >
        <HistoryFilter />
      </div>
      <div
        className="main-content"
        style={{ flex: 1, padding: '16px', display: 'flex', flexDirection: 'column' }}
      >
        <Table style={{ width: '100%', textAlign: 'left' }}>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Operation</TableHead>
              <TableHead>Target</TableHead>
              <TableHead>Result</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>12:00:01</TableCell>
              <TableCell>Modify</TableCell>
              <TableCell>cn=admin</TableCell>
              <TableCell style={{ color: 'green' }}>Success (0)</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
