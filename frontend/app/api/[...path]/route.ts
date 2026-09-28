import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const allowedPaths = new Set([
  "auth/register",
  "auth/login",
  "auth/logout",
  "auth/me",
  "auth/github",
  "repos",
  "articles",
  "github/install",
  "github/setup",
  "github/callback",
]);

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const joinedPath = path.join("/");
  if (!allowedPaths.has(joinedPath)) {
    return Response.json({ error: "not found" }, { status: 404 });
  }

  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const headers = new Headers();
  for (const name of ["content-type", "cookie", "origin", "user-agent", "x-forwarded-for", "x-real-ip"]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  const upstreamURL = new URL(`${baseURL}/api/${joinedPath}`);
  upstreamURL.search = request.nextUrl.search;
  const upstream = await fetch(upstreamURL, {
    method: request.method,
    headers,
    body: request.method === "GET" || request.method === "HEAD" ? undefined : await request.arrayBuffer(),
    cache: "no-store",
    redirect: "manual",
  });

  const responseHeaders = new Headers();
  for (const name of ["content-type", "set-cookie", "retry-after", "location"]) {
    const value = upstream.headers.get(name);
    if (value) responseHeaders.set(name, value);
  }
  const body = upstream.status === 204 || upstream.status === 304 ? null : await upstream.arrayBuffer();
  return new Response(body, {
    status: upstream.status,
    headers: responseHeaders,
  });
}

export const GET = proxy;
export const POST = proxy;
