export interface TokenResponse {
  token: string;
  expires_at: string;
}

export interface RegistrationToken {
  token: string;
  created_at: string;
  expires_at: string;
  used: boolean;
}
