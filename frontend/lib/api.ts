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

export interface Election {
  id: string;
  title: string;
  description: string | null;
  status: string;
  startTime: string;
  endTime: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateElectionInput {
  title: string;
  description: string;
  startTime: string;
  endTime: string;
}

export async function createElection(input: CreateElectionInput): Promise<Election> {
  const response = await apiFetch("/api/elections", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return response.json();
}

export async function listElections(): Promise<Election[]> {
  const response = await apiFetch("/api/elections");
  return response.json();
}

export async function getElection(id: string): Promise<Election> {
  const response = await apiFetch(`/api/elections/${id}`);
  return response.json();
}

export interface Candidate {
  id: string;
  portfolioId: string;
  fullName: string;
  manifesto: string | null;
  createdAt: string;
}

export interface Portfolio {
  id: string;
  electionId: string;
  title: string;
  description: string | null;
  createdAt: string;
  candidates: Candidate[];
}

export async function listPortfolios(electionId: string): Promise<Portfolio[]> {
  const response = await apiFetch(`/api/elections/${electionId}/portfolios`);
  return response.json();
}

export async function createPortfolio(
  electionId: string,
  title: string,
  description: string
): Promise<Portfolio> {
  const response = await apiFetch(`/api/elections/${electionId}/portfolios`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, description }),
  });
  return response.json();
}

export async function deletePortfolio(electionId: string, portfolioId: string): Promise<void> {
  await apiFetch(`/api/elections/${electionId}/portfolios/${portfolioId}`, { method: "DELETE" });
}

export async function createCandidate(
  electionId: string,
  portfolioId: string,
  fullName: string,
  manifesto: string
): Promise<Candidate> {
  const response = await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fullName, manifesto }),
    }
  );
  return response.json();
}

export async function deleteCandidate(
  electionId: string,
  portfolioId: string,
  candidateId: string
): Promise<void> {
  await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates/${candidateId}`,
    { method: "DELETE" }
  );
}
