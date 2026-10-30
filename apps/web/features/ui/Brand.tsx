import Link from "next/link";

export function BrandMark({ decorative = false }: { decorative?: boolean }) {
  return (
    <svg
      className={decorative ? "brand-symbol decorative" : "brand-symbol"}
      viewBox="0 0 48 48"
      aria-hidden="true"
    >
      <path
        d="M18 7h12c8.3 0 14 5.1 14 12.5S38.3 32 30 32h-4v9h-8V7Zm8 7.5v10h4c3.9 0 6-1.8 6-5s-2.1-5-6-5h-4Z"
        fill="currentColor"
        fillRule="evenodd"
      />
      <path
        className="brand-symbol-accent"
        d="M4 12.5a1 1 0 0 1 1.4-.9l8.6 4V41l-8.8-4.1A2 2 0 0 1 4 35.1V12.5Z"
      />
    </svg>
  );
}

export function Brand({ href = "/", className = "" }: { href?: string; className?: string }) {
  return (
    <Link href={href} className={`brand ${className}`.trim()} aria-label="Репет, на главную">
      <BrandMark />
      <span>Репет</span>
    </Link>
  );
}
