import { DatePipe } from '@angular/common';
import { Component, computed, inject, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DeviceStore } from '../../../core/stores/device.store';

@Component({
  selector: 'grc-device-detail',
  templateUrl: './device-detail.html',
  imports: [DatePipe, ButtonModule, RouterLink],
})
export class DeviceDetail {
  private readonly deviceStore = inject(DeviceStore);
  deviceId = input<string | undefined>(undefined);
  device = computed(() => this.deviceStore.deviceById()(this.deviceId() ?? ''));

  isDeviceOnline = computed(() => {
    const lastSeen = this.device()?.last_seen_at;
    if (!lastSeen) return false;
    return Date.now() - new Date(lastSeen).getTime() < 5 * 60 * 1000;
  });
}
