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
  approvedBy: string | null;
  rejectionReason: string | null;
  electorateDatabaseId: string | null;
  electorateFilter: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

export interface CreateElectionInput {
  title: string;
  description: string;
  startTime: string;
  endTime: string;
  electorateDatabaseId: string;
  electorateFilter: Record<string, string>;
}

export async function createElection(input: CreateElectionInput): Promise<Election> {
  const response = await apiFetch("/api/elections", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return response.json();
}

export async function updateElection(
  electionId: string,
  input: CreateElectionInput
): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}`, {
    method: "PUT",
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
  photoUrl: string | null;
  displayOrder: number;
  createdAt: string;
}

export interface Portfolio {
  id: string;
  electionId: string;
  title: string;
  description: string | null;
  displayOrder: number;
  createdAt: string;
  candidates: Candidate[];
}

export interface OrderEntry {
  id: string;
  order: number;
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
  photoUrl: string
): Promise<Candidate> {
  const response = await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fullName, photoUrl }),
    }
  );
  return response.json();
}

export async function updateCandidate(
  electionId: string,
  portfolioId: string,
  candidateId: string,
  fullName: string,
  photoUrl: string
): Promise<Candidate> {
  const response = await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates/${candidateId}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fullName, photoUrl }),
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

export async function requestPasswordReset(email: string): Promise<{ status: string }> {
  const response = await apiFetch("/api/password-reset/request", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  return response.json();
}

export async function confirmPasswordReset(
  token: string,
  newPassword: string
): Promise<{ status: string }> {
  const response = await apiFetch("/api/password-reset/confirm", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token, newPassword }),
  });
  return response.json();
}

export async function disableAccount(id: string): Promise<{ status: string }> {
  const response = await apiFetch(`/api/people/${id}/disable`, { method: "POST" });
  return response.json();
}

export async function reactivateAccount(id: string): Promise<{ status: string }> {
  const response = await apiFetch(`/api/people/${id}/reactivate`, { method: "POST" });
  return response.json();
}

export interface AdminAssignment {
  userId: string;
  name: string;
  email: string;
  assignedAt: string;
}

export async function listAssignedAdmins(electionId: string): Promise<AdminAssignment[]> {
  const response = await apiFetch(`/api/elections/${electionId}/admins`);
  return response.json();
}

export async function assignAdmin(electionId: string, userId: string): Promise<AdminAssignment> {
  const response = await apiFetch(`/api/elections/${electionId}/admins`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ userId }),
  });
  return response.json();
}

export async function unassignAdmin(electionId: string, userId: string): Promise<void> {
  await apiFetch(`/api/elections/${electionId}/admins/${userId}`, { method: "DELETE" });
}

export async function submitForApproval(electionId: string): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}/submit`, { method: "POST" });
  return response.json();
}

export async function approveElection(electionId: string): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}/approve`, { method: "POST" });
  return response.json();
}

export async function rejectElection(electionId: string, reason: string): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}/reject`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  });
  return response.json();
}

export async function deleteElection(electionId: string): Promise<void> {
  await apiFetch(`/api/elections/${electionId}`, { method: "DELETE" });
}

export async function archiveElection(electionId: string): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}/archive`, { method: "POST" });
  return response.json();
}

export async function reorderPortfolios(electionId: string, order: OrderEntry[]): Promise<void> {
  await apiFetch(`/api/elections/${electionId}/portfolios/order`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(order),
  });
}

export async function reorderCandidates(
  electionId: string,
  portfolioId: string,
  order: OrderEntry[]
): Promise<void> {
  await apiFetch(`/api/elections/${electionId}/portfolios/${portfolioId}/candidates/order`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(order),
  });
}

export interface ElectorateDatabase {
  id: string;
  name: string;
  description: string | null;
  columns: string[];
  voterCount: number;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface Voter {
  id: string;
  data: Record<string, string>;
}

export interface UploadResult {
  imported: number;
  skipped: number;
}

export type ElectorateSummary =
  | { linked: false; databaseId: null; databaseName: null; filter: Record<string, string>; eligibleCount: number }
  | {
      linked: true;
      databaseId: string;
      databaseName: string;
      filter: Record<string, string>;
      eligibleCount: number;
    };

export async function createElectorateDatabase(
  name: string,
  description: string
): Promise<ElectorateDatabase> {
  const response = await apiFetch("/api/electorate-databases", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
  });
  return response.json();
}

export async function listElectorateDatabases(): Promise<ElectorateDatabase[]> {
  const response = await apiFetch("/api/electorate-databases");
  return response.json();
}

export async function getElectorateDatabase(id: string): Promise<ElectorateDatabase> {
  const response = await apiFetch(`/api/electorate-databases/${id}`);
  return response.json();
}

export async function deleteElectorateDatabase(id: string): Promise<void> {
  await apiFetch(`/api/electorate-databases/${id}`, { method: "DELETE" });
}

export async function uploadVoters(id: string, file: File): Promise<UploadResult> {
  const formData = new FormData();
  formData.append("file", file);
  const response = await apiFetch(`/api/electorate-databases/${id}/upload`, {
    method: "POST",
    body: formData,
  });
  return response.json();
}

export async function listElectorateVoters(
  id: string,
  filter: Record<string, string> = {}
): Promise<Voter[]> {
  const params = new URLSearchParams(filter);
  const response = await apiFetch(`/api/electorate-databases/${id}/voters?${params.toString()}`);
  return response.json();
}

export async function getDistinctValues(id: string, column: string): Promise<string[]> {
  const params = new URLSearchParams({ column });
  const response = await apiFetch(`/api/electorate-databases/${id}/distinct?${params.toString()}`);
  return response.json();
}

export async function getElectionElectorate(electionId: string): Promise<ElectorateSummary> {
  const response = await apiFetch(`/api/elections/${electionId}/electorate`);
  return response.json();
}
