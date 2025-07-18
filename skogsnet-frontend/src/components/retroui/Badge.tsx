import { cn } from "../../lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import { type HTMLAttributes } from "react";

const badgeVariants = cva("font-semibold", {
  variants: {
    variant: {
      default: "bg-gray-200 text-gray-700 dark:bg-zinc-800 dark:text-gray-200",
      outline: "outline-2 outline-foreground text-foreground dark:outline-gray-200 dark:text-gray-200",
      solid: "bg-foreground text-background dark:bg-gray-200 dark:text-zinc-900",
      surface: "outline-2 bg-primary text-black dark:bg-zinc-700 dark:text-gray-100",
    },
    size: {
      sm: "px-2 py-1 text-xs font-medium",
      md: "px-2.5 py-1.5 text-sm font-medium",
      lg: "px-3 py-2 text-base font-medium",
    },
  },
  defaultVariants: {
    variant: "default",
    size: "md",
  },
});

interface ButtonProps
  extends HTMLAttributes<HTMLSpanElement>,
  VariantProps<typeof badgeVariants> { }

export function Badge({
  children,
  size = "md",
  variant = "default",
  className = "",
  ...props
}: ButtonProps) {
  return (
    <span
      className={cn(badgeVariants({ variant, size }), className)}
      {...props}
    >
      {children}
    </span>
  );
}
