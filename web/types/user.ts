export type User = {
  id: string;
  username: string;
  protected: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
};

export type UserListResponse = {
  count: number;
  page: number;
  page_size: number;
  results: User[];
};

export type ListUsersParams = {
  page?: number;
  page_size?: number;
  search?: string;
};

export type CreateUserPayload = {
  username: string;
  password: string;
};

export type ChangePasswordPayload = {
  password: string;
};

export type SetUserEnabledPayload = {
  enabled: boolean;
};
