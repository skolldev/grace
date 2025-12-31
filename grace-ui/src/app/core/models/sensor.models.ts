export interface SensorInfo {
  sensor_id: string;
  name: string;
  sensor_type: string;
  unit: string;
  source: string;
}

export interface ReportSensorsRequest {
  sensors: SensorInfo[];
}

export interface ReportSensorsResponse {
  status: string;
  count: number;
}

export interface DeviceSensor {
  device_id: string;
  sensor_id: string;
  name: string;
  display_name: string | null;
  sensor_type: string;
  unit: string;
  enabled: boolean;
  source: string;
}

export interface UpdateSensorConfigRequest {
  enabled: string[];
}

export interface SensorConfigResponse {
  enabled: string[];
}
