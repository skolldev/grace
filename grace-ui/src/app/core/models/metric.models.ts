export interface CpuMetrics {
  percent: number;
  cores: number;
}

export interface RamMetrics {
  total_gb: number;
  used_gb: number;
  percent: number;
}

export interface DiskMetrics {
  mount: string;
  total_gb: number;
  used_gb: number;
  percent: number;
}

export interface NetworkMetrics {
  rx_bytes: number;
  tx_bytes: number;
  rx_bytes_per_sec: number;
  tx_bytes_per_sec: number;
}

export interface DefaultMetricData {
  cpu: CpuMetrics;
  ram: RamMetrics;
  disk: DiskMetrics[];
  network: NetworkMetrics;
  uptime_seconds: number;
  sensors?: Record<string, number>;
}

export interface Metric {
  id: number;
  device_id: string;
  timestamp: string;
  data: DefaultMetricData;
}

// Aggregated metrics types (for time-bucketed queries)
export interface AggregateValue {
  avg: number;
  min: number;
  max: number;
}

export interface AggregatedMetricData {
  cpu: { percent: AggregateValue };
  ram: { percent: AggregateValue };
  network: {
    rx_sec: AggregateValue;
    tx_sec: AggregateValue;
  };
}

export interface AggregatedMetric {
  timestamp: string;
  data: AggregatedMetricData;
}
