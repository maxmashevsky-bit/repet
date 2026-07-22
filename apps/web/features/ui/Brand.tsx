import Link from "next/link";

export function BrandMark({ decorative = false }: { decorative?: boolean }) {
  return (
    <svg
      className={decorative ? "brand-symbol decorative" : "brand-symbol"}
      viewBox="0 0 48 56"
      aria-hidden="true"
    >
      <path
        d="M10 48V17C10 9.8 15.8 4 23 4h4c7.7 0 14 6.3 14 14s-6.3 14-14 14H18"
        fill="none"
        stroke="currentColor"
        strokeWidth="8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        className="brand-symbol-accent"
        d="M6 50V36c0-7.2 5.8-13 13-13h22v9H20c-2.8 0-5 2.2-5 5v13H6Z"
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
