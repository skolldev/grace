import { ChangeDetectionStrategy, Component, inject, resource } from '@angular/core';
import { lastValueFrom } from 'rxjs';
import { DevicesService } from '../../core/services/devices.service';
import { DeviceTable } from './device-table/device-table';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-devices',
  templateUrl: './devices.html',
  imports: [DeviceTable],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Devices {
  private readonly devicesService = inject(DevicesService);

  devices = resource({
    params: () => ({}),
    loader: () => lastValueFrom(this.devicesService.getAll()),
  });
}