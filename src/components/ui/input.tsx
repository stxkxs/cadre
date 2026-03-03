import * as React from 'react';
import { cn } from '@/lib/utils';

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          'flex h-10 w-full rounded-md border border-input-border bg-input px-3 py-2 text-sm text-foreground placeholder:text-dim focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0AEFB7]/50 focus-visible:border-[#0AEFB7]/50 disabled:cursor-not-allowed disabled:opacity-50',
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = 'Input';

export { Input };
