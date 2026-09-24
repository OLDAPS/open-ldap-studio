import { Label } from '@/components/ui/label';
import { NativeSelect } from '@/components/ui/native-select';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { AppDialog } from '@/components/ui/app-dialog';
import React from 'react';

export function ReferralChooser() {
  return (
    <AppDialog className="referral-chooser-dialog" title="Follow Referral">
      <div className="modal__header">
        <h3>Follow Referral</h3>
        <Button variant="ghost" size="icon-sm" className="modal__close">
          ×
        </Button>
      </div>
      <div className="modal__body">
        <p>
          The server referred you to <strong>ldap://other.example.org</strong>.
        </p>
        <p>Choose a connection profile to follow this referral:</p>
        <NativeSelect style={{ width: '100%', marginTop: '8px' }}>
          <option>Localhost OpenLDAP</option>
          <option>Create New Profile...</option>
        </NativeSelect>
        <div style={{ marginTop: '16px' }}>
          <Label>
            <Checkbox /> Remember for this session
          </Label>
        </div>
      </div>
      <div className="modal__footer">
        <Button variant="outline">Cancel</Button>
        <Button variant="default">Follow</Button>
      </div>
    </AppDialog>
  );
}
