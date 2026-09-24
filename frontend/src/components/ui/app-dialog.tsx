import type { CSSProperties, ReactNode } from 'react';
import { cn } from 'cn';

import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog';

type AppDialogProps = {
  children: ReactNode;
  className?: string;
  style?: CSSProperties;
  title: string;
};

/**
 * Keeps the desktop dialog geometry while delegating focus, escape handling,
 * the backdrop, and portal behavior to the shadcn dialog primitive.
 */
function AppDialog({ children, className, style, title }: AppDialogProps) {
  return (
    <Dialog open>
      <DialogContent className={cn('modal', className)} showCloseButton={false} style={style}>
        <DialogTitle className="sr-only">{title}</DialogTitle>
        {children}
      </DialogContent>
    </Dialog>
  );
}

export { AppDialog };
