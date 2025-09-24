import { cn } from '../../utils/cn.js';
import { HTMLAttributes, ImgHTMLAttributes, ReactNode } from 'react';

interface AvatarProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

interface AvatarImageProps extends ImgHTMLAttributes<HTMLImageElement> {}

interface AvatarFallbackProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export function Avatar({ className, children, ...props }: AvatarProps) {
  return (
    <div
      className={cn(
        "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export function AvatarImage({ className, ...props }: AvatarImageProps) {
  return (
    <img
      className={cn("aspect-square h-full w-full object-cover", className)}
      {...props}
    />
  );
}

export function AvatarFallback({ className, children, ...props }: AvatarFallbackProps) {
  return (
    <div
      className={cn(
        "flex h-full w-full items-center justify-center rounded-full bg-muted text-sm font-medium",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}
