import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Log } from '../models/api.models';

@Injectable({
  providedIn: 'root',
})
export class LogsService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiUrl}/logs`;

  getAll(): Observable<Log[]> {
    return this.http.get<Log[]>(this.baseUrl);
  }
}
