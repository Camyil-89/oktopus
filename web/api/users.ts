import { fetchApi } from "./base";
import type {
  ChangePasswordPayload,
  CreateUserPayload,
  ListUsersParams,
  SetUserEnabledPayload,
  User,
  UserListResponse,
} from "@/types/user";

function buildListQuery(params: ListUsersParams): string {
  const q = new URLSearchParams();
  if (params.page != null) {
    q.set("page", String(params.page));
  }
  if (params.page_size != null) {
    q.set("page_size", String(params.page_size));
  }
  if (params.search) {
    q.set("search", params.search);
  }
  const qs = q.toString();
  return qs ? `/api/users?${qs}` : "/api/users";
}

export async function listUsers(params: ListUsersParams = {}) {
  return fetchApi<UserListResponse>(buildListQuery(params), "GET");
}

export async function getUser(id: string) {
  return fetchApi<User>(`/api/users/${id}`, "GET");
}

export async function createUser(payload: CreateUserPayload) {
  return fetchApi<User>("/api/users", "POST", payload);
}

export async function changeUserPassword(
  id: string,
  payload: ChangePasswordPayload,
) {
  return fetchApi<void>(`/api/users/${id}/password`, "PATCH", payload);
}

export async function deleteUser(id: string) {
  return fetchApi<void>(`/api/users/${id}`, "DELETE");
}

export async function setUserEnabled(id: string, payload: SetUserEnabledPayload) {
  return fetchApi<User>(`/api/users/${id}/enabled`, "PATCH", payload);
}
