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
  voterVerificationColumns: string[];
  accentColor: string | null;
  accentColorSecondary: string | null;
  accentColorTertiary: string | null;
  logoUrl: string | null;
  ballotStyle: string;
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
  voterVerificationColumns: string[];
  accentColor: string;
  accentColorSecondary: string;
  accentColorTertiary: string;
  logoUrl: string;
  ballotStyle: string;
}

// Every update goes through one full-shape endpoint, but the UI is split
// across several small forms (basics, electorate, settings) that each only
// let an admin change their own slice. This builds a baseline payload from
// the election as it stands today, so a form that edits one slice never
// accidentally blanks out another - a caller overrides only the fields its
// own form actually exposes.
export function electionToCreateInput(election: Election): CreateElectionInput {
  return {
    title: election.title,
    description: election.description ?? "",
    startTime: election.startTime,
    endTime: election.endTime,
    electorateDatabaseId: election.electorateDatabaseId ?? "",
    electorateFilter: election.electorateFilter,
    voterVerificationColumns: election.voterVerificationColumns,
    accentColor: election.accentColor ?? "",
    accentColorSecondary: election.accentColorSecondary ?? "",
    accentColorTertiary: election.accentColorTertiary ?? "",
    logoUrl: election.logoUrl ?? "",
    ballotStyle: election.ballotStyle,
  };
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
  runningMates: string[];
  photoUrl: string | null;
  displayOrder: number;
  createdAt: string;
}

// The ballot's one votable unit may be a joint ticket, not just one
// person - this is the single place that decides how that reads as text,
// so every card/list/result shows a ticket the same way.
export function candidateTicketName(candidate: Pick<Candidate, "fullName" | "runningMates">) {
  return [candidate.fullName, ...candidate.runningMates].join(" & ");
}

// Mirrors the backend's election.BallotStyles - the set of ballot layouts
// an election can choose between. A plain list, not an enum, since this is
// expected to grow as more layouts get designed.
export const BALLOT_STYLES: { value: string; label: string; description: string }[] = [
  {
    value: "compact",
    label: "Compact list",
    description: "A dense row per candidate - small photo, name beside it. Good for long lists.",
  },
  {
    value: "cards",
    label: "Cards",
    description: "Still one per row like compact, but a taller card - a bigger photo, name below it.",
  },
];
export const DEFAULT_BALLOT_STYLE = "compact";

export interface Portfolio {
  id: string;
  electionId: string;
  title: string;
  description: string | null;
  // Every candidate under this portfolio is a fixed two-person ticket
  // (exactly one running mate) when true, or a single person when false -
  // set once at creation, applies to the whole portfolio uniformly.
  runningMateRequired: boolean;
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
  description: string,
  runningMateRequired: boolean
): Promise<Portfolio> {
  const response = await apiFetch(`/api/elections/${electionId}/portfolios`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, description, runningMateRequired }),
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
  photoUrl: string,
  runningMates: string[] = []
): Promise<Candidate> {
  const response = await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fullName, photoUrl, runningMates }),
    }
  );
  return response.json();
}

export async function updateCandidate(
  electionId: string,
  portfolioId: string,
  candidateId: string,
  fullName: string,
  photoUrl: string,
  runningMates: string[] = []
): Promise<Candidate> {
  const response = await apiFetch(
    `/api/elections/${electionId}/portfolios/${portfolioId}/candidates/${candidateId}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fullName, photoUrl, runningMates }),
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

export interface ScheduleChange {
  id: string;
  electionId: string;
  actorUserId: string;
  actorName: string;
  previousEndTime: string;
  newEndTime: string;
  reason: string;
  createdAt: string;
}

export async function updateElectionSchedule(
  electionId: string,
  endTime: string,
  reason: string
): Promise<Election> {
  const response = await apiFetch(`/api/elections/${electionId}/schedule`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ endTime, reason }),
  });
  return response.json();
}

export async function listScheduleChanges(electionId: string): Promise<ScheduleChange[]> {
  const response = await apiFetch(`/api/elections/${electionId}/schedule-changes`);
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
  sensitiveColumns: string[];
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

export async function uploadVoters(
  id: string,
  file: File,
  sensitiveColumns: string[] = []
): Promise<UploadResult> {
  const formData = new FormData();
  formData.append("file", file);
  if (sensitiveColumns.length > 0) {
    formData.append("sensitiveColumns", sensitiveColumns.join(","));
  }
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

export interface IssueVoterCredentialsResult {
  eligible: number;
  issued: number;
  alreadyIssued: number;
  invites: { email: string; link: string }[];
}

export async function issueVoterCredentials(
  electionId: string
): Promise<IssueVoterCredentialsResult> {
  const response = await apiFetch(`/api/elections/${electionId}/voter-credentials/issue`, {
    method: "POST",
  });
  return response.json();
}

export interface CandidateResult {
  candidateId: string;
  fullName: string;
  photoUrl: string | null;
  runningMates: string[];
  voteCount: number;
}

export interface PortfolioResult {
  portfolioId: string;
  title: string;
  candidates: CandidateResult[];
  // Set only for an uncontested portfolio (exactly one candidate/ticket) -
  // a deliberate "No" on its Yes/No ballot, kept separate from
  // abstainCount.
  noCount: number | null;
  abstainCount: number;
  totalBallots: number;
}

export interface ElectionResults {
  electionId: string;
  portfolios: PortfolioResult[];
}

export async function getElectionResults(electionId: string): Promise<ElectionResults> {
  const response = await apiFetch(`/api/elections/${electionId}/results`);
  return response.json();
}

// --- Voter-facing (public, no dashboard session) ---

export interface VoteElectionSummary {
  id: string;
  title: string;
  description: string | null;
  startTime: string;
  endTime: string;
  accentColor: string | null;
  accentColorSecondary: string | null;
  accentColorTertiary: string | null;
  logoUrl: string | null;
  ballotStyle: string;
}

export interface VoteTokenStatus {
  status: "needs_verification" | "authenticated" | "voted" | "locked" | "invalid";
  election?: VoteElectionSummary;
  verificationColumns?: string[];
}

export async function getVoteTokenStatus(token: string): Promise<VoteTokenStatus> {
  const response = await apiFetch(`/api/vote/${token}`);
  return response.json();
}

export async function authenticateVoter(
  token: string,
  verification: Record<string, string>
): Promise<void> {
  await apiFetch(`/api/vote/${token}/authenticate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(verification),
  });
}

export interface VoterBallot {
  election: VoteElectionSummary;
  portfolios: Portfolio[];
}

export async function getVoterBallot(electionId: string): Promise<VoterBallot> {
  const params = new URLSearchParams({ electionId });
  const response = await apiFetch(`/api/vote/ballot?${params.toString()}`);
  return response.json();
}

export async function castVote(
  electionId: string,
  selections: Record<string, string | null>,
  explicitNo: Record<string, boolean> = {}
): Promise<void> {
  await apiFetch(`/api/vote/cast`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ electionId, selections, explicitNo }),
  });
}

// --- Public pulse (turnout transparency, no login) ---

export interface TurnoutPoint {
  bucketStart: string;
  count: number;
}

export interface PortfolioAbstention {
  portfolioId: string;
  title: string;
  uncontested: boolean;
  ballots: number | null;
  abstentions: number | null;
}

export interface PulseScheduleChange {
  previousEndTime: string;
  newEndTime: string;
  reason: string;
  createdAt: string;
}

export interface PulseSnapshot {
  election: {
    id: string;
    title: string;
    description: string | null;
    status: string;
    startTime: string;
    endTime: string;
    accentColor: string | null;
    accentColorSecondary: string | null;
    accentColorTertiary: string | null;
    logoUrl: string | null;
  };
  eligibleCount: number;
  votedCount: number;
  turnoutSeries: TurnoutPoint[];
  portfolioAbstention: PortfolioAbstention[];
  scheduleChanges: PulseScheduleChange[];
  resultsPublished: boolean;
  results?: PortfolioResult[];
  resultsPublishedAt?: string;
  refreshedAt: string;
}

export async function getElectionPulse(electionId: string): Promise<PulseSnapshot> {
  const response = await apiFetch(`/api/elections/${electionId}/pulse`);
  return response.json();
}

export async function publishResults(electionId: string): Promise<ElectionResults> {
  const response = await apiFetch(`/api/elections/${electionId}/publish-results`, {
    method: "POST",
  });
  return response.json();
}

// --- Branding (system-wide default, public read) ---

export interface BrandingSettings {
  accentColor: string | null;
  accentColorSecondary: string | null;
  accentColorTertiary: string | null;
  logoUrl: string | null;
  updatedAt: string;
}

export async function getBranding(): Promise<BrandingSettings> {
  const response = await apiFetch("/api/settings/branding");
  return response.json();
}

export async function updateBranding(
  accentColor: string,
  accentColorSecondary: string,
  accentColorTertiary: string,
  logoUrl: string
): Promise<BrandingSettings> {
  const response = await apiFetch("/api/settings/branding", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ accentColor, accentColorSecondary, accentColorTertiary, logoUrl }),
  });
  return response.json();
}

// --- File uploads (candidate photos, logos) ---

export type UploadKind = "candidate-photos" | "logos";

export interface PresignedUpload {
  uploadUrl: string;
  publicUrl: string;
}

async function presignUpload(kind: UploadKind, contentType: string): Promise<PresignedUpload> {
  const response = await apiFetch("/api/uploads/presign", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ kind, contentType }),
  });
  return response.json();
}

export interface AuditLogEntry {
  seq: number;
  actorUserId: string | null;
  actorName: string | null;
  actorEmail: string | null;
  action: string;
  metadata: Record<string, unknown>;
  ipAddress: string | null;
  userAgent: string | null;
  createdAt: string;
}

export async function listAuditLog(
  filter: { before?: number; limit?: number; electionId?: string } = {}
): Promise<AuditLogEntry[]> {
  const params = new URLSearchParams();
  if (filter.before !== undefined) params.set("before", String(filter.before));
  if (filter.limit !== undefined) params.set("limit", String(filter.limit));
  if (filter.electionId) params.set("electionId", filter.electionId);
  const response = await apiFetch(`/api/audit-log?${params.toString()}`);
  const body = await response.json();
  return body.entries;
}

export interface AuditChainVerification {
  ok: boolean;
  entries: number;
  brokenAtSeq?: number;
}

export async function verifyAuditLog(): Promise<AuditChainVerification> {
  const response = await apiFetch("/api/audit-log/verify");
  return response.json();
}

// uploadFile gets a presigned URL from our backend, then PUTs the file
// straight to R2 with a plain fetch (no credentials, no API base URL -
// it's a different origin entirely) so the file itself never passes
// through our own server.
export async function uploadFile(kind: UploadKind, file: File): Promise<string> {
  const { uploadUrl, publicUrl } = await presignUpload(kind, file.type);
  const response = await fetch(uploadUrl, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });
  if (!response.ok) {
    throw new ApiError(response.status, "Upload failed. Please try again.");
  }
  return publicUrl;
}
