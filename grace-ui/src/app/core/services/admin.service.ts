import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { RegistrationToken, TokenResponse } from '../models/api.models';

@Injectable({
  providedIn: 'root',
})
export class AdminService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/admin`;

  createToken(): Observable<TokenResponse> {
    return this.http.post<TokenResponse>(`${this.baseUrl}/tokens`, {});
  }

  getTokens(): Observable<RegistrationToken[]> {
    return this.http.get<RegistrationToken[]>(`${this.baseUrl}/tokens`);
  }
}
