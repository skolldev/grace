import { Component, computed, effect, inject, input, resource, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { firstValueFrom } from 'rxjs';
import { DeviceSensor, SensorsService } from '../../core';
import { DeviceStore } from '../../core/stores/device.store';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-sensor-overview',
  templateUrl: './sensor-overview.html',
  imports: [
    FormsModule,
    TableModule,
    TagModule,
    ButtonModule,
    RouterLink,
    IconFieldModule,
    InputIconModule,
    InputTextModule,
    SelectModule,
  ],
})
export class SensorOverview {
  private readonly sensorsService = inject(SensorsService);
  private readonly deviceStore = inject(DeviceStore);

  readonly statusOptions = [
    { label: 'Enabled', value: true },
    { label: 'Disabled', value: false },
  ];

  readonly typeOptions = [
    { label: 'Temperature', value: 'temperature' },
    { label: 'Usage', value: 'usage' },
    { label: 'Current', value: 'current' },
    { label: 'Voltage', value: 'voltage' },
    { label: 'Power', value: 'power' },
    { label: 'Clock', value: 'clock' },
    { label: 'Fan', value: 'fan' },
    { label: 'Other', value: 'other' },
  ];

  deviceId = input<string>();

  device = computed(() => this.deviceStore.deviceById()(this.deviceId() ?? ''));

  private sensorsResource = resource({
    params: () => ({ deviceId: this.deviceId() }),
    loader: ({ params }) =>
      params.deviceId
        ? firstValueFrom(this.sensorsService.getDeviceSensors(params.deviceId))
        : Promise.resolve([]),
  });

  private localSensors = signal<DeviceSensor[]>([]);

  sensors = computed<DeviceSensor[]>(() => this.localSensors());

  loading = computed(() => this.sensorsResource.isLoading());

  updating = signal<boolean>(false);

  constructor() {
    effect(() => {
      if (this.sensorsResource.hasValue()) {
        this.localSensors.set(this.sensorsResource.value());
      }
    });
  }

  toggleSensor(sensor: DeviceSensor): void {
    const deviceId = this.deviceId();
    if (!deviceId) return;

    this.updating.set(true);

    const newEnabledState = !sensor.enabled;
    const currentSensors = this.localSensors();

    const enabledSensorIds = currentSensors
      .filter((s) => (s.sensor_id === sensor.sensor_id ? newEnabledState : s.enabled))
      .map((s) => s.sensor_id);

    this.sensorsService.updateSensorConfig(deviceId, { enabled: enabledSensorIds }).subscribe({
      next: () => {
        this.localSensors.update((sensors) =>
          sensors.map((s) =>
            s.sensor_id === sensor.sensor_id ? { ...s, enabled: newEnabledState } : s
          )
        );
        this.updating.set(false);
      },
      error: () => {
        this.updating.set(false);
      },
    });
  }

  getEnabledSeverity(enabled: boolean): 'success' | 'danger' {
    return enabled ? 'success' : 'danger';
  }

  getEnabledLabel(enabled: boolean): string {
    return enabled ? 'Enabled' : 'Disabled';
  }
}
