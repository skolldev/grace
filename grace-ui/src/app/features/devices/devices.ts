import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { DeviceStore } from '../../core/stores/device.store';
import { DeviceDetail } from './device-detail/device-detail';
import { DeviceTable } from './device-table/device-table';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-devices',
  templateUrl: './devices.html',
  imports: [DeviceTable, DeviceDetail],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Devices {
  private readonly deviceStore = inject(DeviceStore);

  devices = this.deviceStore.devices;
  selectedDeviceId = signal<string | undefined>(undefined);
}
