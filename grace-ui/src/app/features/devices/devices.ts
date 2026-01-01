import { ChangeDetectionStrategy, Component, effect, inject, resource, signal } from '@angular/core';
import { interval, lastValueFrom } from 'rxjs';
import { DeviceSummary } from '../../core/models';
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
  tableData = signal<DeviceSummary[]>([]);
  countSig = signal(0);
  
  constructor() {

    interval(5000).subscribe(() => {
      this.countSig.update(prev => prev + 1);
    });
    

    effect(() => {
      const data = this.devices.value();
      if (data !== undefined) {
        this.tableData.set(data);
      }
    });
  }

  devices = resource({
    params: () => ({ count: this.countSig() }),
    loader: () => lastValueFrom(this.devicesService.getAll()),
    
  });
}