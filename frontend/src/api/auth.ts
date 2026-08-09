import type { AuthResponse, User } from "../types/api";
import { apiRequest } from "./client";

interface Credentials {
  email: string;
  password: string;
}

export const authApi = {
  login(credentials: Credentials): Promise<AuthResponse> {
    return apiRequest<AuthResponse>("/login", {
      method: "POST",
      body: JSON.stringify(credentials),
    });
  },

  register(credentials: Credentials): Promise<AuthResponse> {
    return apiRequest<AuthResponse>("/register", {
      method: "POST",
      body: JSON.stringify(credentials),
    });
  },

  me(): Promise<User> {
    return apiRequest<User>("/me");
  },
};
