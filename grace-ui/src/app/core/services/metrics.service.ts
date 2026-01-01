import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Metric } from '../models';

export interface MetricsQueryParams {
  start?: string;
  end?: string;
  limit?: number;
}

@Injectable({
  providedIn: 'root',
})
export class MetricsService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/devices`;

  getByDeviceId(deviceId: string, params?: MetricsQueryParams): Observable<Metric[]> {
    let httpParams = new HttpParams();

    if (params?.start) {
      httpParams = httpParams.set('start', params.start);
    }
    if (params?.end) {
      httpParams = httpParams.set('end', params.end);
    }
    if (params?.limit !== undefined) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }

    return this.http.get<Metric[]>(`${this.baseUrl}/${deviceId}/metrics`, { params: httpParams });
  }
}
