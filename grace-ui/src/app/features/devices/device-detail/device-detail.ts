import { DatePipe } from '@angular/common';
import { Component, computed, inject, input, resource, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { Select } from 'primeng/select';
import { firstValueFrom } from 'rxjs';
import {
  AggregatedSensorData,
  AggregatedSensorDataPoint,
  AggregatedSensorMetricsResponse,
} from '../../../core/models';
import {
  AggregatedMetricsParams,
  MetricsService,
  Resolution,
} from '../../../core/services/metrics.service';
import { DeviceStore } from '../../../core/stores/device.store';
import { ChartCard } from '../chart-card/chart-card';

interface SensorChart {
  sensor_id: string;
  title: string;
  type: string;
  unit: string;
  data: { label: string; data: { avg: number | null; min: number | null; max: number | null } }[];
}

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
  private readonly router = inject(Router);
  deviceId = input<string | undefined>(undefined);
  device = computed(() => this.deviceStore.deviceById()(this.deviceId() ?? ''));

  hasSensors = computed(() => this.device()?.has_sensors ?? false);

  isOnDetailPage = computed(() => {
    const deviceId = this.deviceId();
    return deviceId ? this.router.url.includes(`/devices/${deviceId}`) : false;
  });

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
      deviceId: this.deviceId(),
      start: this.dateRange().start,
      end: this.dateRange().end,
      resolution: this.timeframe().resolution,
    }),
    loader: ({ params }) =>
      firstValueFrom(
        this.metricsService.getByDeviceId(params.deviceId!, params as AggregatedMetricsParams)
      ),
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

  private bytesToMegabits(bytes: number | null): number | null {
    return bytes !== null ? (bytes * 8) / 1000000 : null;
  }

  networkRxMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource.value().map((metric) => ({
          label: metric.timestamp,
          data: {
            avg: this.bytesToMegabits(metric.data.network.rx_sec.avg),
            min: this.bytesToMegabits(metric.data.network.rx_sec.min),
            max: this.bytesToMegabits(metric.data.network.rx_sec.max),
          },
        }))
      : []
  );

  networkTxMetric = computed(() =>
    this.metricsResource.hasValue()
      ? this.metricsResource.value().map((metric) => ({
          label: metric.timestamp,
          data: {
            avg: this.bytesToMegabits(metric.data.network.tx_sec.avg),
            min: this.bytesToMegabits(metric.data.network.tx_sec.min),
            max: this.bytesToMegabits(metric.data.network.tx_sec.max),
          },
        }))
      : []
  );

  sensorMetricsResource = resource({
    params: () => ({
      deviceId: this.deviceId(),
      start: this.dateRange().start,
      end: this.dateRange().end,
      resolution: this.timeframe().resolution,
      hasSensors: this.hasSensors(),
    }),
    loader: ({ params }): Promise<AggregatedSensorMetricsResponse> =>
      params.hasSensors && params.deviceId
        ? firstValueFrom(
            this.metricsService.getSensorMetrics(params.deviceId, params as AggregatedMetricsParams)
          )
        : Promise.resolve({ sensors: [] }),
  });

  sensorCharts = computed((): SensorChart[] => {
    if (!this.sensorMetricsResource.hasValue()) return [];

    return this.sensorMetricsResource.value().sensors.map((sensor: AggregatedSensorData) => ({
      sensor_id: sensor.sensor_id,
      title: `${sensor.name} (${sensor.unit})`,
      type: sensor.sensor_type,
      unit: sensor.unit,
      data: sensor.data.map((point: AggregatedSensorDataPoint) => ({
        label: point.timestamp,
        data: point.value,
      })),
    }));
  });
}
