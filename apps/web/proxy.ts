import { NextRequest, NextResponse } from "next/server";

const publicRoutes = ["/login", "/register", "/forgot-password", "/reset-password", "/terms", "/privacy"];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  if (publicRoutes.some((route) => pathname.startsWith(route))) {
    return NextResponse.next();
  }
  const hasSession = request.cookies.has("repet_session");
  if (!hasSession) {
    const loginURL = new URL("/login", request.url);
    loginURL.searchParams.set("returnTo", pathname);
    return NextResponse.redirect(loginURL);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/", "/messages/:path*", "/calendar/:path*", "/tasks/:path*", "/settings/:path*"]
};
