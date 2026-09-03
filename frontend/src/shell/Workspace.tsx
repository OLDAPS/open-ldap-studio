/**
 * The document area every perspective shares: tabs on top, the view's own
 * content in the middle, the progress-and-logs panel underneath.
 *
 * It lives apart from Shell so a view can use it without importing the module
 * that imports the view.
 */
import type { ReactNode } from 'react';

import { BottomPanel } from './BottomPanel';
import { DocumentTabs } from './DocumentTabs';

export function Workspace({ children }: { children: ReactNode }) {
  return (
    <div className="main">
      <DocumentTabs />
      {children}
      <BottomPanel />
    </div>
  );
}
