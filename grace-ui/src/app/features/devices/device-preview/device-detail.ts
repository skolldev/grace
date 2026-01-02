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

interface Timeframe {
  timeframe: string;
  resolution: Resolution;
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

  timeframe = signal<Timeframe>({ timeframe: '1h', resolution: '1m' });

  dateRange = computed(() => {
    const end = new Date();
    const start = new Date();
    const timeframe = this.timeframe().timeframe;

    const value = parseInt(timeframe.slice(0, -1));
    const unit = timeframe.slice(-1);

    switch (unit) {
      case 'h':
        start.setHours(start.getHours() - value);
        break;
      case 'd':
        start.setDate(start.getDate() - value);
        break;
    }

    return {
      start: start.toISOString(),
      end: end.toISOString(),
    };
  });

  timeframeConfig: { label: string; value: Timeframe }[] = [
    { label: 'Last hour', value: { timeframe: '1h', resolution: '1m' } },
    { label: 'Last 24 hours', value: { timeframe: '24h', resolution: '1h' } },
    { label: 'Last 7 days', value: { timeframe: '7d', resolution: '6h' } },
    { label: 'Last 30 days', value: { timeframe: '30d', resolution: '1d' } },
  ];

  metricsResource = resource({
    params: () => ({
      start: this.dateRange().start,
      end: this.dateRange().end,
      resolution: this.timeframe().resolution,
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
          .map((metric) => ({
            label: metric.timestamp,
            data: {
              avg: metric.data.network.rx_sec.avg !== null ? (metric.data.network.rx_sec.avg * 8) / 1000000 : null,
              min: metric.data.network.rx_sec.min !== null ? (metric.data.network.rx_sec.min * 8) / 1000000 : null,
              max: metric.data.network.rx_sec.max !== null ? (metric.data.network.rx_sec.max * 8) / 1000000 : null,
            },
          }))
      : []
  );

  networkTxMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource
          .value()
          .map((metric) => ({
            label: metric.timestamp,
            data: {
              avg: metric.data.network.tx_sec.avg !== null ? (metric.data.network.tx_sec.avg * 8) / 1000000 : null,
              min: metric.data.network.tx_sec.min !== null ? (metric.data.network.tx_sec.min * 8) / 1000000 : null,
              max: metric.data.network.tx_sec.max !== null ? (metric.data.network.tx_sec.max * 8) / 1000000 : null,
            },
          }))
      : []
  );
}
