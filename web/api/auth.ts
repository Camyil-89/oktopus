import { fetchApi } from "./base";
import type { AuthUser, LoginPayload } from "@/types/auth";

export async function getMe() {
  return fetchApi<AuthUser>("/api/auth/me", "GET");
}

export async function login(payload: LoginPayload) {
  return fetchApi<AuthUser>("/api/auth/login", "POST", payload);
}

export async function logout() {
  return fetchApi<void>("/api/auth/logout", "POST");
}
