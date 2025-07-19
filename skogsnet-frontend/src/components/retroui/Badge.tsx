import { cn } from "../../lib/utils";
import { cva, type VariantProps } from "class-variance-authority";
import { type HTMLAttributes } from "react";

const backgroundColorDark = "#557188";
const backgroundColorLight = "#FF0000";

const badgeVariants = cva("font-semibold", {
  variants: {
    variant: {
      default: `bg-[${backgroundColorLight}] text-gray-700 dark:bg-[${backgroundColorDark}] dark:text-gray-200`,
      outline: `outline-2 outline-foreground text-foreground dark:outline-[${backgroundColorDark}] dark:text-gray-200`,
      solid: `bg-[${backgroundColorLight}] text-background dark:bg-[${backgroundColorDark}] dark:text-zinc-900`,
      surface: `outline-2 bg-primary text-black dark:bg-[${backgroundColorDark}] dark:text-gray-100`,
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
