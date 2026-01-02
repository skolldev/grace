import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { AggregatedMetric } from '../models';

export type Resolution = '1m' | '5m' | '15m' | '1h' | '6h' | '1d';

export interface AggregatedMetricsParams {
  start: string;
  end: string;
  resolution: Resolution;
}

@Injectable({
  providedIn: 'root',
})
export class MetricsService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/devices`;

  getByDeviceId(deviceId: string, params: AggregatedMetricsParams): Observable<AggregatedMetric[]> {
    const httpParams = new HttpParams()
      .set('start', params.start)
      .set('end', params.end)
      .set('resolution', params.resolution);

    return this.http.get<AggregatedMetric[]>(`${this.baseUrl}/${deviceId}/metrics`, {
      params: httpParams,
    });
  }
}
