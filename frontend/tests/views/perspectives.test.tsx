/**
 * Every perspective mounts, and mounts with the shell around it.
 *
 * The six rail entries are the whole application's surface; a view that throws
 * on mount takes the window with it, and there is no route to fall back to.
 * This is the cheapest possible guard against that, and against a perspective
 * quietly losing the tabs or the logs panel it is supposed to share.
 */
import { render, screen, cleanup } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';

import { CommandProvider } from '@/commands/CommandProvider';
import { Shell } from '@/shell/Shell';
import type { Perspective } from '@/store/session';
import { useSession } from '@/store/session';

const PERSPECTIVES: Perspective[] = [
  'connections',
  'browser',
  'searches',
  'schema',
  'files',
  'preferences',
];

afterEach(cleanup);

describe('perspectives', () => {
  it.each(PERSPECTIVES)('%s mounts inside the shell', (perspective) => {
    useSession.setState({ perspective });

    const { container } = render(
      <CommandProvider>
        <Shell />
      </CommandProvider>,
    );

    // The chrome the wireframe puts on every screen.
    expect(container.querySelector('.rail')).not.toBeNull();
    expect(container.querySelector('.statusbar')).not.toBeNull();
    expect(screen.getByLabelText('Progress and logs')).toBeInTheDocument();

    // …and the perspective's own sidebar plus document area.
    expect(container.querySelector('.sidebar')).not.toBeNull();
    expect(container.querySelector('.main')).not.toBeNull();
  });
});
