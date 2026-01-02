import { DatePipe } from '@angular/common';
import { Component, computed, inject, input, resource, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { Select } from 'primeng/select';
import { firstValueFrom } from 'rxjs';
import {
  AggregatedMetricsParams,
  MetricsService,
  Resolution,
} from '../../../core/services/metrics.service';
import { DeviceStore } from '../../../core/stores/device.store';
import { ChartCard } from '../chart-card/chart-card';
interface TimeframeOption {
  label: string;
  value: number;
}

interface ResolutionOption {
  label: string;
  value: Resolution;
}

@Component({
  selector: 'grc-device-detail',
  templateUrl: './device-detail.html',
  imports: [DatePipe, ButtonModule, RouterLink, Select, FormsModule, ChartCard],
})
export class DeviceDetail {
  private readonly deviceStore = inject(DeviceStore);
  private readonly metricsService = inject(MetricsService);
  deviceId = input<string | undefined>(undefined);
  device = computed(() => this.deviceStore.deviceById()(this.deviceId() ?? ''));

  isDeviceOnline = computed(() => {
    const lastSeen = this.device()?.last_seen_at;
    if (!lastSeen) return false;
    return Date.now() - new Date(lastSeen).getTime() < 5 * 60 * 1000;
  });

  resolution = signal<Resolution>('5m');
  timeframe = signal<number>(1);

  dateRange = computed(() => {
    const end = new Date();
    const start = new Date();
    const days = this.timeframe();

    start.setDate(end.getDate() - days);

    return {
      start: start.toISOString(),
      end: end.toISOString(),
    };
  });

  resolutionOptions: ResolutionOption[] = [
    { label: '1 minute', value: '1m' },
    { label: '5 minutes', value: '5m' },
    { label: '15 minutes', value: '15m' },
    { label: '1 hour', value: '1h' },
    { label: '6 hours', value: '6h' },
    { label: '1 day', value: '1d' },
  ];

  timeframeOptions: TimeframeOption[] = [
    { label: '1 day', value: 1 },
    { label: '7 days', value: 7 },
    { label: '14 days', value: 14 },
    { label: '30 days', value: 30 },
    { label: '90 days', value: 90 },
  ];

  metricsResource = resource({
    params: () => ({
      start: this.dateRange().start,
      end: this.dateRange().end,
      resolution: this.resolution(),
    }),
    loader: ({ params }: { params: AggregatedMetricsParams }) =>
      firstValueFrom(this.metricsService.getByDeviceId(this.deviceId() ?? '', params)),
  });

  cpuMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource
          .value()
          .map((metric) => ({ label: metric.timestamp, data: metric.data.cpu.percent }))
      : []
  );

  ramMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource
          .value()
          .map((metric) => ({ label: metric.timestamp, data: metric.data.ram.percent }))
      : []
  );

  networkRxMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource
          .value()
          .map((metric) => ({ label: metric.timestamp, data: metric.data.network.rx_sec }))
      : []
  );

  networkTxMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource
          .value()
          .map((metric) => ({ label: metric.timestamp, data: metric.data.network.tx_sec }))
      : []
  );
}
