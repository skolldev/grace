import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { Device } from '../../../core/models/device.models';

@Component({
  host: {
    class: 'flex flex-col flex-1 overflow-hidden',
  },
  selector: 'grc-device-table',
  templateUrl: './device-table.html',
  imports: [DatePipe, TableModule, TagModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DeviceTable {
  devices = input.required<Device[]>();
  loading = input(false);

}
