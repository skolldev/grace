import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'logs',
    loadComponent: () => import('./features/logs/logs').then((m) => m.Logs),
  },
  {
    path: 'devices',
    loadComponent: () => import('./features/devices/devices').then((m) => m.Devices),
  },
  {
    path: 'devices/:deviceId',
    loadComponent: () =>
      import('./features/devices/device-detail/device-detail').then((m) => m.DeviceDetail),
  },
  {
    path: 'devices/:deviceId/sensors',
    loadComponent: () => import('./features/sensors/sensor-overview').then((m) => m.SensorOverview),
  },
];
