/**
 * The wizard actually saves, and the password does not travel with the profile.
 *
 * The second half is the one worth a test: the Go decoder refuses a profile
 * payload carrying a secret, so a frontend that put one there would fail at
 * runtime in the one place a user cannot work around. This pins the split.
 */
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ConnectionWizard } from '@/features/connections/components/ConnectionWizard';

const saveProfile = vi.fn();
const storeProfileSecret = vi.fn();
const testConnection = vi.fn();

vi.mock('@/bridge/client', () => ({
  isEmbedded: () => true,
  bridge: {
    saveProfile: (...args: unknown[]) => saveProfile(...args),
    storeProfileSecret: (...args: unknown[]) => storeProfileSecret(...args),
    testConnection: (...args: unknown[]) => testConnection(...args),
  },
}));

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
  it('saves the profile and files the secret through a separate call', async () => {
    const user = userEvent.setup();
    const onSaved = vi.fn();
    render(<ConnectionWizard onClose={() => {}} onSaved={onSaved} />);

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
    render(<ConnectionWizard onClose={() => {}} />);

    await user.type(screen.getByLabelText('Hostname'), 'localhost');
    await user.click(screen.getByRole('button', { name: '2 · Authentication' }));
    await user.click(screen.getByRole('button', { name: 'Anonymous' }));
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    await waitFor(() => expect(saveProfile).toHaveBeenCalledTimes(1));
    expect(storeProfileSecret).not.toHaveBeenCalled();
  });

  it('checks the connection without saving anything', async () => {
    const user = userEvent.setup();
    render(<ConnectionWizard onClose={() => {}} />);

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
    render(<ConnectionWizard onClose={() => {}} />);

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
    render(<ConnectionWizard onClose={() => {}} />);

    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Check authentication' }));

    expect(await screen.findByText(/result 49/)).toBeInTheDocument();
    expect(screen.getByText(/invalid credentials/)).toBeInTheDocument();
  });

  it('surfaces a save failure instead of closing', async () => {
    saveProfile.mockRejectedValue(new Error('connections.json is read-only'));

    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<ConnectionWizard onClose={onClose} />);

    await fillNetworkAndAuth(user);
    await user.click(screen.getByRole('button', { name: 'Finish' }));

    expect(await screen.findByText('connections.json is read-only')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });
});
