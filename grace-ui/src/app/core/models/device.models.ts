import { DefaultMetricData } from './metric.models';

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
  has_sensors: boolean;
}

export interface LatestMetric {
  timestamp: string;
  data: DefaultMetricData;
}

export interface DeviceSummary extends Device {
  latest_metrics: LatestMetric | null;
}
