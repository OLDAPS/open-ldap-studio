import { Button } from '@/components/ui/button';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function CertificateTrustDialog() {
  return (
    <AppDialog className="certificate-trust-dialog" title="Untrusted Certificate">
      <div className="modal__header">
        <h3>Untrusted Certificate</h3>
      </div>
      <div className="modal__body">
        <p>The server presented a certificate that is not trusted by the OS or the application.</p>
        <div
          style={{
            backgroundColor: 'var(--bg-inset)',
            padding: '16px',
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '0.85rem',
          }}
        >
          Subject: CN=ldap.example.org,O=Example
          <br />
          Issuer: CN=Example Root CA,O=Example
          <br />
          Fingerprint: AA:BB:CC:DD...
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Reject</Button>
        <Button variant="outline">Trust Once</Button>
        <Button variant="default">Trust Permanently</Button>
      </div>
    </AppDialog>
  );
}
