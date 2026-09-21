/**
 * The connection wizard's network step actually drives the rest of the wizard.
 *
 * These are the three things that were silently inert: the transport select
 * held no state, the port could be nudged by the mouse, and step 2's clear-text
 * warning was derived from a local that nothing could change — so it claimed a
 * clear-text bind even over LDAPS.
 */
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it } from 'vitest';

import { ConnectionWizard } from '@/features/connections/components/ConnectionWizard';

afterEach(cleanup);

const port = () => screen.getByLabelText('Port') as HTMLInputElement;
const encryption = () => screen.getByLabelText('Encryption') as HTMLSelectElement;

describe('connection wizard — network step', () => {
  it('moves the port with the transport until the user names one', async () => {
    const user = userEvent.setup();
    render(<ConnectionWizard onClose={() => {}} />);

    expect(port().value).toBe('389');

    await user.selectOptions(encryption(), 'ldaps');
    expect(encryption().value).toBe('ldaps');
    expect(port().value).toBe('636');

    // Once the port is the user's, the transport stops overwriting it.
    await user.clear(port());
    await user.type(port(), '10389');
    await user.selectOptions(encryption(), 'startTLS');
    expect(port().value).toBe('10389');
  });

  it('is typed rather than nudged: no spinner, digits only', async () => {
    const user = userEvent.setup();
    render(<ConnectionWizard onClose={() => {}} />);

    // A number input is what carries spinner buttons and wheel increments.
    expect(port().getAttribute('type')).toBe('text');
    expect(port().getAttribute('inputMode')).toBe('numeric');

    await user.clear(port());
    await user.type(port(), '6a3!6');
    expect(port().value).toBe('636');
  });

  it('warns about a clear-text bind only when the transport is unencrypted', async () => {
    const user = userEvent.setup();
    render(<ConnectionWizard onClose={() => {}} />);

    // Default is StartTLS, so step 2 must not cry wolf.
    await user.click(screen.getByRole('button', { name: '2 · Authentication' }));
    expect(screen.queryByText('Clear text')).toBeNull();

    await user.click(screen.getByRole('button', { name: '1 · Network' }));
    await user.selectOptions(encryption(), 'none');
    expect(screen.getByText('Unencrypted')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '2 · Authentication' }));
    expect(screen.getByText('Clear text')).toBeInTheDocument();

    // Anonymous has no credential to expose.
    await user.click(screen.getByRole('button', { name: 'Anonymous' }));
    expect(screen.queryByText('Clear text')).toBeNull();
  });
});
