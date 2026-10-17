const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function parseErrorMessage(response: Response): Promise<string> {
  try {
    const body = await response.json();
    if (typeof body?.error === "string") return body.error;
  } catch {
    // response wasn't JSON; fall through to a generic message
  }
  return `Request failed with status ${response.status}`;
}

async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  // credentials: "include" always - the session cookie only ever matters
  // for auth-related calls, but sending it everywhere means no future
  // endpoint can silently forget it.
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
  });
  if (!response.ok) {
    throw new ApiError(response.status, await parseErrorMessage(response));
  }
  return response;
}

export async function getBootstrapStatus(): Promise<{ available: boolean }> {
  const response = await apiFetch("/api/bootstrap/status");
  return response.json();
}

export interface InitiateBootstrapInput {
  name: string;
  email: string;
  password: string;
  secret: string;
}

export async function initiateBootstrap(
  input: InitiateBootstrapInput
): Promise<{ status: string }> {
  const response = await apiFetch("/api/bootstrap/initiate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return response.json();
}

export async function confirmBootstrap(token: string): Promise<{ status: string }> {
  const response = await apiFetch(
    `/api/bootstrap/confirm?token=${encodeURIComponent(token)}`
  );
  return response.json();
}

export interface SessionUser {
  id: string;
  name: string;
  email: string;
  role: string;
}

export async function login(email: string, password: string): Promise<SessionUser> {
  const response = await apiFetch("/api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  return response.json();
}

export async function logout(): Promise<void> {
  await apiFetch("/api/logout", { method: "POST" });
}

export async function getSession(): Promise<SessionUser> {
  const response = await apiFetch("/api/session");
  return response.json();
}

export type InvitableRole = "SYSTEMADMIN" | "ADMIN" | "APPROVER";

export async function createInvite(
  name: string,
  email: string,
  role: InvitableRole
): Promise<{ status: string }> {
  const response = await apiFetch("/api/invite", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, email, role }),
  });
  return response.json();
}

export async function acceptInvite(
  token: string,
  password: string
): Promise<{ status: string }> {
  const response = await apiFetch("/api/invite/accept", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token, password }),
  });
  return response.json();
}

export interface Person {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  createdAt: string;
}

export async function listPeople(): Promise<Person[]> {
  const response = await apiFetch("/api/people");
  return response.json();
}

export async function revokeInvite(id: string): Promise<{ status: string }> {
  const response = await apiFetch(`/api/people/${id}`, { method: "DELETE" });
  return response.json();
}
