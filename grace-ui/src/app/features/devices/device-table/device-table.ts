import { DatePipe } from '@angular/common';
import { Component, input } from '@angular/core';
import { ProgressBar } from 'primeng/progressbar';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { CpuMetrics, DeviceSummary, DiskMetrics } from '../../../core/models';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-device-table',
  templateUrl: './device-table.html',
  imports: [DatePipe, ProgressBar, TableModule, TagModule],
})
export class DeviceTable {
  devices = input.required<DeviceSummary[]>();

  getCPUPercent(cpu: CpuMetrics | undefined): number {
    if (cpu === undefined) return 0;
    return this.roundWithPrecision(cpu.percent, 2);
  }

  getTotalDiskPercent(disks: DiskMetrics[] | undefined): number {
    if (!disks || disks.length === 0) return 0;
    const totalUsed = disks.reduce((sum, d) => sum + d.used_gb, 0);
    const totalSize = disks.reduce((sum, d) => sum + d.total_gb, 0);
    return this.roundWithPrecision((totalSize > 0 ? (totalUsed / totalSize) * 100 : 0), 2);
  }

  formatSpeed(bytesPerSec: number | undefined): string {
    if (bytesPerSec === undefined || bytesPerSec === 0) return '0';
    const mbPerSec = bytesPerSec / (1024 * 1024);
    if (mbPerSec < 0.01) return '0';
    return mbPerSec.toFixed(2);
  }

  private roundWithPrecision(value: number, precision: number): number {
    const multiplier = 10 ** precision;
    return Math.round(value * multiplier) / multiplier;
  }
}
