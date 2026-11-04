/**
 * The wizard actually saves, and the password does not travel with the profile.
 *
 * The second half is the one worth a test: the Go decoder refuses a profile
 * payload carrying a secret, so a frontend that put one there would fail at
 * runtime in the one place a user cannot work around. This pins the split.
 */
import { act, cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { fakeApplicationClient } from '../../helpers/applicationClient';

import { ConnectionWizard } from '@/features/connections/components/ConnectionWizard';

const saveProfile = vi.fn();
const storeProfileSecret = vi.fn();
const testConnection = vi.fn();

const client = fakeApplicationClient({
  isAvailable: () => true,
  saveProfile,
  storeProfileSecret,
  testConnection,
});
const renderWizard = (props: Parameters<typeof ConnectionWizard>[0]) =>
  render(
    <ApplicationClientProvider client={client}>
      <ConnectionWizard {...props} />
    </ApplicationClientProvider>,
  );

beforeEach(() => {
  saveProfile.mockReset().mockResolvedValue({ id: 'p1', name: 'local-dev' });
  storeProfileSecret.mockReset().mockResolvedValue(undefined);
  testConnection.mockReset().mockResolvedValue({
    reachable: true,
    encrypted: true,
    tlsVerified: true,
    bound: true,
    boundDn: 'cn=admin,dc=example,dc=org',
    vendorName: 'OpenLDAP',
    namingContexts: ['dc=example,dc=org'],
    result: { code: 0, matchedDn: '', diagnosticMessage: '', interpretation: '' },
  });
});

afterEach(cleanup);

async function fillNetworkAndAuth(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Hostname'), 'localhost');
  await user.clear(screen.getByLabelText('Port'));
  await user.type(screen.getByLabelText('Port'), '1389');

  await user.click(screen.getByRole('button', { name: '2 · Authentication' }));
  await user.type(screen.getByLabelText('Bind DN'), 'cn=admin,dc=example,dc=org');
  await user.type(screen.getByLabelText('Password'), 'adminpassword');
}

describe('connection wizard — saving', () => {
  it('discards a check result after the hostname changes', async () => {
    let resolve!: (value: unknown) => void;
    testConnection.mockReturnValue(
      new Promise((done) => {
        resolve = done;
      }),
    );
    const user = userEvent.setup();
    renderWizard({ onClose: () => {} });
    await user.type(screen.getByLabelText('Hostname'), 'old.example.org');
    await user.click(screen.getByRole('button', { name: 'Check network parameter' }));
    await user.clear(screen.getByLabelText('Hostname'));
    await user.type(screen.getByLabelText('Hostname'), 'new.example.org');
    await act(async () =>
      resolve({
        reachable: true,
        encrypted: true,
        tlsVerified: true,
        bound: false,
        namingContexts: ['dc=old,dc=org'],
        result: { code: 0 },
      }),
    );
    expect(screen.queryByText(/✓ reachable/)).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '3 · Browser options' }));
    expect(screen.getByLabelText('Base DN')).toHaveValue('');
  });

  it('saves the profile and files the secret through a separate call', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    renderWizard({ onClose: () => {}, onSaved });

    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    await waitFor(() => expect(saveProfile).toHaveBeenCalledTimes(1));

    const [payload] = saveProfile.mock.calls[0] as [Record<string, unknown>];
    expect(payload.host).toBe('localhost');
    expect(payload.port).toBe(1389);
    expect(payload.bindDn).toBe('cn=admin,dc=example,dc=org');

    // The whole point: no key of the profile payload carries the password.
    expect(JSON.stringify(payload)).not.toContain('adminpassword');

    expect(storeProfileSecret).toHaveBeenCalledWith('p1', 'adminpassword');
    expect(onSaved).toHaveBeenCalledWith('p1');
  });

  it('does not file a secret for an anonymous bind', async () => {
    const user = userEvent.setup();
    renderWizard({ onClose: () => {} });

    await user.type(screen.getByLabelText('Hostname'), 'localhost');
    await user.click(screen.getByRole('button', { name: '2 · Authentication' }));
    await user.click(screen.getByRole('button', { name: 'Anonymous' }));
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    await waitFor(() => expect(saveProfile).toHaveBeenCalledTimes(1));
    expect(storeProfileSecret).not.toHaveBeenCalled();
  });

  it('checks the connection without saving anything', async () => {
    const user = userEvent.setup();
    renderWizard({ onClose: () => {} });

    await user.type(screen.getByLabelText('Hostname'), 'localhost');
    await user.click(screen.getByRole('button', { name: 'Check network parameter' }));

    await waitFor(() => expect(testConnection).toHaveBeenCalledTimes(1));
    expect(saveProfile).not.toHaveBeenCalled();
    expect(storeProfileSecret).not.toHaveBeenCalled();

    // What the server said comes back onto the screen. The naming context
    // appears more than once by design — once as what the server reported, and
    // once as the base DN now offered — so this asserts on the report.
    expect(await screen.findByText(/OpenLDAP/)).toBeInTheDocument();
    expect(screen.getByText(/reachable/)).toBeInTheDocument();
    expect(screen.getAllByText(/dc=example,dc=org/).length).toBeGreaterThan(0);
  });

  it('offers the base DN the server reported', async () => {
    const user = userEvent.setup();
    renderWizard({ onClose: () => {} });

    await user.type(screen.getByLabelText('Hostname'), 'localhost');
    await user.click(screen.getByRole('button', { name: 'Check network parameter' }));
    await waitFor(() => expect(testConnection).toHaveBeenCalled());

    await user.click(screen.getByRole('button', { name: '3 · Browser options' }));
    expect((screen.getByLabelText('Base DN') as HTMLInputElement).value).toBe('dc=example,dc=org');
  });

  it('reports a rejected bind with the server’s own result code', async () => {
    testConnection.mockResolvedValue({
      reachable: true,
      encrypted: false,
      tlsVerified: false,
      bound: false,
      result: {
        code: 49,
        matchedDn: '',
        diagnosticMessage: 'invalid credentials',
        interpretation: 'the bind DN or password was not accepted',
      },
    });

    const user = userEvent.setup();
    renderWizard({ onClose: () => {} });

    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Check authentication' }));

    expect(await screen.findByText(/result 49/)).toBeInTheDocument();
    expect(screen.getByText(/invalid credentials/)).toBeInTheDocument();
  });

  it('surfaces a save failure instead of closing', async () => {
    saveProfile.mockRejectedValue(new Error('connections.json is read-only'));

    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWizard({ onClose });

    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    expect(await screen.findByText('connections.json is read-only')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('reuses the saved profile when credential storage fails and Finish is retried', async () => {
    let created = 0;
    saveProfile.mockImplementation(async (profile: Record<string, unknown>) => ({
      ...profile,
      id: profile.id ?? `p${++created}`,
    }));
    storeProfileSecret.mockRejectedValueOnce(new Error('credential store is locked'));

    const user = userEvent.setup();
    const onSaved = vi.fn();
    const onClose = vi.fn();
    renderWizard({ onClose, onSaved });
    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    expect(await screen.findByText('credential store is locked')).toBeInTheDocument();
    expect(onSaved).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByLabelText('Password')).toHaveValue('adminpassword');

    // Edits after the failed attempt must keep the identity of the saved draft.
    await user.clear(screen.getByLabelText('Bind DN'));
    await user.type(screen.getByLabelText('Bind DN'), 'cn=reader,dc=example,dc=org');
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(saveProfile).toHaveBeenCalledTimes(2);
    expect(saveProfile.mock.calls[1]?.[0]).toMatchObject({
      id: 'p1',
      bindDn: 'cn=reader,dc=example,dc=org',
    });
    expect(created).toBe(1);
    expect(storeProfileSecret.mock.calls).toEqual([
      ['p1', 'adminpassword'],
      ['p1', 'adminpassword'],
    ]);
    expect(onSaved).toHaveBeenCalledExactlyOnceWith('p1');
    expect(screen.getByLabelText('Password')).toHaveValue('');
  });
});
