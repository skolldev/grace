import { DatePipe } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { Log, LogsService } from '../../core';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-logs',
  templateUrl: './logs.html',
  imports: [DatePipe, TableModule, TagModule, CardModule],
})
export class Logs implements OnInit {
  private readonly logsService = inject(LogsService);

  logs = signal<Log[]>([]);
  loading = signal(true);

  ngOnInit(): void {
    this.logsService.getAll().subscribe({
      next: (logs) => {
        this.logs.set(logs);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      },
    });
  }

  getSeverity(type: Log['type']): 'success' | 'warn' | 'danger' {
    switch (type) {
      case 'info':
        return 'success';
      case 'warning':
        return 'warn';
      case 'error':
        return 'danger';
    }
  }
}
