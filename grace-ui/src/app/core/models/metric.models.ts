export interface MetricsPayload {
  device_id: string;
  timestamp?: string;
  metrics: Record<string, unknown>;
}

export interface MetricsResponse {
  status: string;
}

export interface Metric {
  id: number;
  device_id: string;
  timestamp: string;
  data: Record<string, unknown>;
}
