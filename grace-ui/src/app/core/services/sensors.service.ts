import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import {
  DeviceSensor,
  ReportSensorsRequest,
  ReportSensorsResponse,
  SensorConfigResponse,
  UpdateSensorConfigRequest,
} from '../models/api.models';

@Injectable({
  providedIn: 'root',
})
export class SensorsService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = environment.apiUrl;

  reportSensors(deviceId: string, request: ReportSensorsRequest): Observable<ReportSensorsResponse> {
    return this.http.post<ReportSensorsResponse>(
      `${this.baseUrl}/devices/${deviceId}/sensors`,
      request
    );
  }

  getDeviceSensors(deviceId: string): Observable<DeviceSensor[]> {
    return this.http.get<DeviceSensor[]>(`${this.baseUrl}/devices/${deviceId}/sensors`);
  }

  updateSensorConfig(
    deviceId: string,
    request: UpdateSensorConfigRequest
  ): Observable<SensorConfigResponse> {
    return this.http.put<SensorConfigResponse>(
      `${this.baseUrl}/devices/${deviceId}/sensors/config`,
      request
    );
  }

  getSensorConfig(deviceId: string): Observable<SensorConfigResponse> {
    return this.http.get<SensorConfigResponse>(
      `${this.baseUrl}/devices/${deviceId}/sensors/config`
    );
  }
}
