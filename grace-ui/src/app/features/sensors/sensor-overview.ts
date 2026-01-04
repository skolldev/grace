import { Component, computed, inject, input, resource } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { InputTextModule } from 'primeng/inputtext';
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
    TableModule,
    TagModule,
    ButtonModule,
    RouterLink,
    IconFieldModule,
    InputIconModule,
    InputTextModule,
  ],
})
export class SensorOverview {
  private readonly sensorsService = inject(SensorsService);
  private readonly deviceStore = inject(DeviceStore);

  deviceId = input<string>();

  device = computed(() => this.deviceStore.deviceById()(this.deviceId() ?? ''));

  sensorsResource = resource({
    params: () => ({ deviceId: this.deviceId() }),
    loader: ({ params }) =>
      params.deviceId
        ? firstValueFrom(this.sensorsService.getDeviceSensors(params.deviceId))
        : Promise.resolve([]),
  });

  sensors = computed<DeviceSensor[]>(() =>
    this.sensorsResource.hasValue() ? this.sensorsResource.value() : []
  );

  loading = computed(() => this.sensorsResource.isLoading());

  getEnabledSeverity(enabled: boolean): 'success' | 'danger' {
    return enabled ? 'success' : 'danger';
  }

  getEnabledLabel(enabled: boolean): string {
    return enabled ? 'Enabled' : 'Disabled';
  }
}
