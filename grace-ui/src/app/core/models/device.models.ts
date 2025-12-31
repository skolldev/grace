export interface RegisterRequest {
  token: string;
  hostname: string;
  os: string;
  arch: string;
  ip_address?: string;
}

export interface RegisterResponse {
  device_id: string;
  message: string;
}

export interface Device {
  id: string;
  hostname: string;
  os: string;
  arch: string;
  ip_address: string | null;
  registered_at: string;
  last_seen_at: string;
}

export interface DeviceWithMetrics extends Device {
  latest_metrics: Record<string, unknown> | null;
}
