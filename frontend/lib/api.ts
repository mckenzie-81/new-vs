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

export async function getBootstrapStatus(): Promise<{ available: boolean }> {
  const response = await fetch(`${API_BASE_URL}/api/bootstrap/status`);
  if (!response.ok) {
    throw new ApiError(response.status, await parseErrorMessage(response));
  }
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
  const response = await fetch(`${API_BASE_URL}/api/bootstrap/initiate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    throw new ApiError(response.status, await parseErrorMessage(response));
  }
  return response.json();
}

export async function confirmBootstrap(token: string): Promise<{ status: string }> {
  const response = await fetch(
    `${API_BASE_URL}/api/bootstrap/confirm?token=${encodeURIComponent(token)}`
  );
  if (!response.ok) {
    throw new ApiError(response.status, await parseErrorMessage(response));
  }
  return response.json();
}
