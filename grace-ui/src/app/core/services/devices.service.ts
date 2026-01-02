import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { DeviceSummary, RegisterRequest, RegisterResponse } from '../models';

@Injectable({
  providedIn: 'root',
})
export class DevicesService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/devices`;

  register(request: RegisterRequest): Observable<RegisterResponse> {
    return this.http.post<RegisterResponse>(`${this.baseUrl}/register`, request);
  }

  getAll(): Observable<DeviceSummary[]> {
    return this.http.get<DeviceSummary[]>(this.baseUrl);
  }

  getById(id: string): Observable<DeviceSummary> {
    return this.http.get<DeviceSummary>(`${this.baseUrl}/${id}`);
  }

  delete(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`);
  }
}
