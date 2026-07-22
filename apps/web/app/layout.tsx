import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "Репет",
  description: "Спокойный кабинет для занятий, заданий и общения"
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
