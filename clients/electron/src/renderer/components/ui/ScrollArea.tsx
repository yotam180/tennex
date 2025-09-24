import { forwardRef, HTMLAttributes, ReactNode } from 'react';
import { cn } from '../../utils/cn.js';

interface ScrollAreaProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn('relative overflow-hidden', className)}
        {...props}
      >
        <div
          className="h-full w-full overflow-auto"
          data-radix-scroll-area-viewport
        >
          {children}
        </div>
      </div>
    );
  }
);

ScrollArea.displayName = 'ScrollArea';
