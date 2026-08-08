import { tokenStore } from "./tokenStore";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

interface ApiErrorBody {
  error?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const token = tokenStore.get();

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  });

  if (response.status === 401) {
    tokenStore.clear();
    if (token && window.location.pathname !== "/login") {
      window.location.replace("/login");
    }
  }

  if (!response.ok) {
    let message = `Ошибка запроса: ${response.status}`;
    try {
      const body = (await response.json()) as ApiErrorBody;
      if (body.error) {
        message = body.error;
      }
    } catch {
      // Код ответа остается запасным сообщением, если backend вернул не JSON.
    }
    throw new ApiError(message, response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
