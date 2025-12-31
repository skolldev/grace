import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { DeviceSensor, ReportSensorsRequest, ReportSensorsResponse, SensorConfigResponse, UpdateSensorConfigRequest } from '../models';

@Injectable({
  providedIn: 'root',
})
export class SensorsService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/devices`;

  reportSensors(deviceId: string, request: ReportSensorsRequest): Observable<ReportSensorsResponse> {
    return this.http.post<ReportSensorsResponse>(
      `${this.baseUrl}/${deviceId}/sensors`,
      request
    );
  }

  getDeviceSensors(deviceId: string): Observable<DeviceSensor[]> {
    return this.http.get<DeviceSensor[]>(`${this.baseUrl}/${deviceId}/sensors`);
  }

  updateSensorConfig(
    deviceId: string,
    request: UpdateSensorConfigRequest
  ): Observable<SensorConfigResponse> {
    return this.http.put<SensorConfigResponse>(
      `${this.baseUrl}/${deviceId}/sensors/config`,
      request
    );
  }

  getSensorConfig(deviceId: string): Observable<SensorConfigResponse> {
    return this.http.get<SensorConfigResponse>(
      `${this.baseUrl}${deviceId}/sensors/config`
    );
  }
}
