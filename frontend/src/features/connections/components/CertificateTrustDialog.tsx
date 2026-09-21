import React from 'react';

export function CertificateTrustDialog() {
  return (
    <div className="modal-overlay">
      <div className="modal certificate-trust-dialog">
        <div className="modal__header">
          <h3>Untrusted Certificate</h3>
        </div>
        <div className="modal__body">
          <p>The server presented a certificate that is not trusted by the OS or the application.</p>
          <div style={{ backgroundColor: 'var(--bg-inset)', padding: '16px', borderRadius: '4px', fontFamily: 'monospace', fontSize: '0.85rem' }}>
            Subject: CN=ldap.example.org,O=Example<br/>
            Issuer: CN=Example Root CA,O=Example<br/>
            Fingerprint: AA:BB:CC:DD...
          </div>
        </div>
        <div className="modal__footer">
          <button className="button">Reject</button>
          <button className="button">Trust Once</button>
          <button className="button button--primary">Trust Permanently</button>
        </div>
      </div>
    </div>
  );
}
