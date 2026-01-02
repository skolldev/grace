import { computed, inject } from '@angular/core';
import {
  patchState,
  signalStore,
  withComputed,
  withHooks,
  withMethods,
  withState,
} from '@ngrx/signals';
import { rxMethod } from '@ngrx/signals/rxjs-interop';
import { interval, pipe, switchMap, tap } from 'rxjs';
import { DeviceSummary } from '../models';
import { DevicesService } from '../services/devices.service';

interface DeviceState {
  devices: DeviceSummary[];
  loading: boolean;
  lastUpdated: Date | null;
  autoRefresh: boolean;
  autoRefreshInterval: number;
}

const initialState: DeviceState = {
  devices: [],
  loading: false,
  lastUpdated: null,
  autoRefresh: true,
  autoRefreshInterval: 5000,
};

export const DeviceStore = signalStore(
  { providedIn: 'root' },
  withState(initialState),
  withComputed((state) => ({
    deviceCount: computed(() => state.devices().length),
  })),
  withMethods((store, devicesService = inject(DevicesService)) => ({
    // Lookup by ID from cache
    deviceById: computed(() => (id: string) => store.devices().find((d) => d.id === id)),
    setAutoRefresh: (autoRefresh: boolean) => {
      patchState(store, { autoRefresh });
    },
    setAutoRefreshInterval: (autoRefreshInterval: number) => {
      patchState(store, { autoRefreshInterval });
    },
    // Load all devices
    loadDevices: rxMethod<void>(
      pipe(
        tap(() => patchState(store, { loading: true })),
        switchMap(() => devicesService.getAll()),
        tap((devices) =>
          patchState(store, {
            devices,
            loading: false,
            lastUpdated: new Date(),
          })
        )
      )
    ),
  })),
  withHooks({
    onInit: (store) => {
      store.loadDevices();
      if (store.autoRefresh()) {
        interval(store.autoRefreshInterval()).subscribe(() => {
          store.loadDevices();
        });
      }
    },
  })
);
