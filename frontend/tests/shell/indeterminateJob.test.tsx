import { act, cleanup, render, renderHook, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, expect, it } from 'vitest';
import { ApplicationClientProvider } from '@/app/ApplicationClientProvider';
import { CommandProvider } from '@/app/CommandProvider';
import { useSession } from '@/app/session';
import { useShellViewModel } from '@/app/useShellViewModel';
import { BottomPanel } from '@/shell/BottomPanel';
import { eventClient } from '../helpers/eventClient';

beforeEach(() =>
  useSession.setState({ jobs: {}, notices: [], panelOpen: true, panelTab: 'progress' }),
);
afterEach(cleanup);

it('announces an uncertain write and shows its error in the progress table', () => {
  const events = eventClient({ isAvailable: () => false });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <ApplicationClientProvider client={events.client}>
      <CommandProvider>{children}</CommandProvider>
    </ApplicationClientProvider>
  );
  renderHook(useShellViewModel, { wrapper });
  act(() =>
    events.emit('job:started', {
      jobId: 'job-1',
      kind: 'commit',
      mode: 'execute',
      profileId: 'p1',
      total: 1,
    }),
  );
  act(() =>
    events.emit('job:finished', {
      jobId: 'job-1',
      state: 'indeterminate',
      summary: 'indeterminate — 1 uncertain',
      error: 'modify: the server may or may not have applied the write',
    }),
  );
  expect(useSession.getState().jobs['job-1']?.state).toBe('indeterminate');
  expect(useSession.getState().notices.at(-1)).toMatchObject({
    level: 'error',
    message: 'Write outcome unknown; cancellation does not establish rollback',
  });
  render(
    <ApplicationClientProvider client={events.client}>
      <BottomPanel />
    </ApplicationClientProvider>,
  );
  expect(
    screen.getByText('modify: the server may or may not have applied the write'),
  ).toBeInTheDocument();
  expect(screen.getByText('indeterminate')).toBeInTheDocument();
});
