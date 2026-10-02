import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function CredentialsModal() {
  return (
    <AppDialog className="credentials-modal" title="Credentials" style={{ width: '600px' }}>
      <div className="modal__header">
        <h3>Credentials</h3>
        <Button variant="ghost" size="icon-sm" className="modal__close">
          ×
        </Button>
      </div>
      <div className="modal__body" style={{ display: 'flex', gap: '16px' }}>
        {/* List of credentials */}
        <div style={{ flex: '1', borderRight: '1px solid var(--border)', paddingRight: '16px' }}>
          <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
            <li
              style={{
                padding: '8px',
                borderBottom: '1px solid var(--border-subtle)',
                background: 'var(--bg-inset)',
              }}
            >
              <strong>Admin Credential</strong> <br />
              <small style={{ color: 'var(--text-muted)' }}>Simple Bind</small>
            </li>
          </ul>
        </div>

        {/* Details */}
        <div style={{ flex: '2' }}>
          <h4>Admin Credential</h4>
          <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
            Assigned to: <strong>Localhost OpenLDAP</strong>
          </p>
          <div style={{ marginTop: '16px' }}>
            <Button variant="outline" style={{ marginRight: '8px' }}>
              Test Bind
            </Button>
            <Button variant="outline">Delete</Button>
          </div>

          <div style={{ marginTop: '32px', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
            Note: Secret held in OS keychain.
          </div>
        </div>
      </div>
    </AppDialog>
  );
}
