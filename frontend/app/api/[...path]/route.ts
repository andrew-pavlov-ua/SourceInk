import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

const allowedPaths = new Set([
  "auth/register",
  "auth/login",
  "auth/email",
  "auth/logout",
  "auth/me",
  "auth/github",
  "repos",
  "articles",
  "published-articles",
  "github/install",
  "github/connect",
  "github/setup",
  "github/callback",
  "github/webhook",
]);

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const articleSlugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function isAllowedPath(path: string[]) {
  const joinedPath = path.join("/");
  return allowedPaths.has(joinedPath)
    || (path.length === 2 && path[0] === "published-articles" && articleSlugPattern.test(path[1]))
    || (path.length === 3 && path[0] === "articles" && uuidPattern.test(path[1]) && path[2] === "publish")
    || (path.length === 3
      && path[0] === "published-articles"
      && uuidPattern.test(path[1])
      && (path[2] === "review" || path[2] === "reviews"));
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const joinedPath = path.join("/");
  if (!isAllowedPath(path)) {
    return Response.json({ error: "not found" }, { status: 404 });
  }

  const baseURL = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080";
  const headers = new Headers();
  for (const name of [
    "content-type",
    "cookie",
    "origin",
    "user-agent",
    "x-forwarded-for",
    "x-real-ip",
    "x-github-delivery",
    "x-github-event",
    "x-hub-signature-256",
  ]) {
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
export const PUT = proxy;
export const DELETE = proxy;
